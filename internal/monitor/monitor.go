// Package monitor 提供SSH登录监控功能
package monitor

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yourusername/ssh_fb/internal/config"
	"github.com/yourusername/ssh_fb/internal/notification"
	"github.com/yourusername/ssh_fb/pkg/firewall"
	"github.com/yourusername/ssh_fb/pkg/ipinfo"
)

// Monitor 结构体封装了SSH监控功能
type Monitor struct {
	config         *config.Config                // 配置信息
	logger         *logrus.Logger               // 日志记录器
	telegram       *notification.Telegram       // Telegram通知器
	firewall       *firewall.UFW                // 防火墙管理器
	ipInfo         *ipinfo.Client               // IP信息查询客户端
	failedAttempts map[string]int               // IP失败尝试次数记录
	bannedIPs      map[string]time.Time         // 被封禁IP及其解封时间
	mu             sync.RWMutex                 // 并发控制锁
}

// NewMonitor 创建并初始化一个新的监控器实例
// 参数:
//   - config: 配置信息
//   - logger: 日志记录器
//   - telegram: Telegram通知器
// 返回:
//   - *Monitor: 初始化后的监控器实例
func NewMonitor(config *config.Config, logger *logrus.Logger, telegram *notification.Telegram) *Monitor {
	return &Monitor{
		config:         config,
		logger:         logger,
		telegram:       telegram,
		firewall:       firewall.NewUFW(),
		ipInfo:         ipinfo.NewClient(config.IPInfo.APIURL, config.IPInfo.Language, config.IPInfo.Timeout, config.IPInfo.RetryCount, config.IPInfo.RetryInterval),
		failedAttempts: make(map[string]int),
		bannedIPs:      make(map[string]time.Time),
	}
}

// Start 启动监控器
// 加载黑名单并开始监控SSH日志
// 返回:
//   - error: 启动过程中的错误信息
func (m *Monitor) Start() error {
	// 加载黑名单
	if err := m.loadBlacklist(); err != nil {
		return err
	}

	// 启动清理协程
	go m.cleanupBannedIPs()

	// 监控SSH日志
	return m.monitorSSHLogs()
}

// loadBlacklist 从文件加载黑名单
// 返回:
//   - error: 加载过程中的错误信息
func (m *Monitor) loadBlacklist() error {
	file, err := os.OpenFile(m.config.Blacklist.File, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ip := scanner.Text()
		if ip != "" {
			m.bannedIPs[ip] = time.Now().Add(time.Duration(m.config.SSHProtection.BanDurationHours) * time.Hour)
		}
	}

	return scanner.Err()
}

// saveBlacklist 保存黑名单到文件
// 返回:
//   - error: 保存过程中的错误信息
func (m *Monitor) saveBlacklist() error {
	file, err := os.OpenFile(m.config.Blacklist.File, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	m.mu.RLock()
	for ip := range m.bannedIPs {
		if _, err := file.WriteString(ip + "\n"); err != nil {
			return err
		}
	}
	m.mu.RUnlock()

	return nil
}

// cleanupBannedIPs 定期清理过期的封禁IP
// 每小时检查一次，解除已过期的IP封禁
func (m *Monitor) cleanupBannedIPs() {
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		m.mu.Lock()
		for ip, banTime := range m.bannedIPs {
			if time.Now().After(banTime) {
				if err := m.firewall.UnbanIP(ip); err != nil {
					m.logger.WithError(err).WithField("ip", ip).Error("解除IP封禁失败")
				} else {
					m.logger.WithField("ip", ip).Info("IP已解除封禁")
				}
				delete(m.bannedIPs, ip)
				delete(m.failedAttempts, ip)
			}
		}
		m.mu.Unlock()
	}
}

// monitorSSHLogs 监控SSH日志文件
// 实时读取并分析SSH日志，处理登录成功和失败事件
// 返回:
//   - error: 监控过程中的错误信息
func (m *Monitor) monitorSSHLogs() error {
	file, err := os.Open(m.config.SSHProtection.SSHLogFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// 移动到文件末尾
	file.Seek(0, 2)

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return err
		}

		if strings.Contains(line, "Failed password") {
			re := regexp.MustCompile(`from (\d+\.\d+\.\d+\.\d+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				m.handleFailedLogin(matches[1])
			}
		} else if strings.Contains(line, "Accepted password") {
			re := regexp.MustCompile(`from (\d+\.\d+\.\d+\.\d+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				m.handleSuccessfulLogin(matches[1])
			}
		}
	}
}

// handleFailedLogin 处理登录失败事件
// 参数:
//   - ip: 登录失败的IP地址
func (m *Monitor) handleFailedLogin(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isIPBanned(ip) {
		m.logger.WithField("ip", ip).Warn("尝试登录的IP已被封禁")
		return
	}

	m.failedAttempts[ip]++
	
	m.logger.WithFields(logrus.Fields{
		"ip":           ip,
		"attempts":     m.failedAttempts[ip],
		"max_attempts": m.config.SSHProtection.MaxFailedAttempts,
	}).Warn("SSH登录失败")

	if m.failedAttempts[ip] >= m.config.SSHProtection.MaxFailedAttempts {
		m.banIP(ip)
	}

	ipInfo := m.ipInfo.FormatIPInfo(ip)
	server := fmt.Sprintf("%s (%s)", m.config.Service.ServiceName, m.config.Service.InstallPath)
	m.telegram.NotifyLoginFailed(ip, ipInfo, server, m.failedAttempts[ip], m.config.SSHProtection.MaxFailedAttempts)
}

// handleSuccessfulLogin 处理登录成功事件
// 参数:
//   - ip: 登录成功的IP地址
func (m *Monitor) handleSuccessfulLogin(ip string) {
	m.logger.WithField("ip", ip).Info("SSH登录成功")

	ipInfo := m.ipInfo.FormatIPInfo(ip)
	server := fmt.Sprintf("%s (%s)", m.config.Service.ServiceName, m.config.Service.InstallPath)
	m.telegram.NotifyLoginSuccess(ip, ipInfo, server)
}

// banIP 封禁指定的IP地址
// 参数:
//   - ip: 要封禁的IP地址
func (m *Monitor) banIP(ip string) {
	banTime := time.Now().Add(time.Duration(m.config.SSHProtection.BanDurationHours) * time.Hour)
	m.bannedIPs[ip] = banTime

	if err := m.firewall.BanIP(ip); err != nil {
		m.logger.WithError(err).WithField("ip", ip).Error("封禁IP失败")
		return
	}

	m.saveBlacklist()

	m.logger.WithFields(logrus.Fields{
		"ip":           ip,
		"duration":     m.config.SSHProtection.BanDurationHours,
		"expire_time": banTime.Format("2006-01-02 15:04:05"),
	}).Info("IP已被封禁")

	ipInfo := m.ipInfo.FormatIPInfo(ip)
	server := fmt.Sprintf("%s (%s)", m.config.Service.ServiceName, m.config.Service.InstallPath)
	m.telegram.NotifyIPBanned(ip, ipInfo, server, time.Duration(m.config.SSHProtection.BanDurationHours)*time.Hour, banTime)
}

// isIPBanned 检查IP是否被封禁
// 参数:
//   - ip: 要检查的IP地址
// 返回:
//   - bool: true表示被封禁，false表示未被封禁
func (m *Monitor) isIPBanned(ip string) bool {
	if banTime, exists := m.bannedIPs[ip]; exists {
		if time.Now().Before(banTime) {
			return true
		}
		delete(m.bannedIPs, ip)
		delete(m.failedAttempts, ip)
		return false
	}
	return false
} 
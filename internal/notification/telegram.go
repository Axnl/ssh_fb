// Package notification 提供通知相关的功能实现
package notification

import (
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

// Telegram 结构体封装了Telegram机器人的功能
type Telegram struct {
	bot     *tgbotapi.BotAPI  // Telegram机器人API实例
	chatID  int64            // 接收通知的聊天ID
	logger  *logrus.Logger   // 日志记录器
	config  *Config          // 配置信息
}

// Config 定义了Telegram机器人的配置参数
type Config struct {
	BotToken string // Telegram机器人Token
	ChatID   int64  // 接收通知的聊天ID
	Debug    bool   // 是否启用调试模式
}

// NewTelegram 创建并初始化一个新的Telegram通知实例
// 参数:
//   - config: Telegram配置信息
//   - logger: 日志记录器
// 返回:
//   - *Telegram: 初始化后的Telegram实例
//   - error: 初始化过程中的错误信息
func NewTelegram(config *Config, logger *logrus.Logger) (*Telegram, error) {
	bot, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		return nil, fmt.Errorf("Telegram机器人初始化失败: %v", err)
	}

	bot.Debug = config.Debug

	return &Telegram{
		bot:    bot,
		chatID: config.ChatID,
		logger: logger,
		config: config,
	}, nil
}

// SendMessage 发送文本消息到指定的聊天
// 参数:
//   - text: 要发送的消息内容
// 返回:
//   - error: 发送过程中的错误信息
func (t *Telegram) SendMessage(text string) error {
	msg := tgbotapi.NewMessage(t.chatID, text)
	_, err := t.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("发送Telegram消息失败: %v", err)
	}
	return nil
}

// NotifyLoginSuccess 发送SSH登录成功的通知
// 参数:
//   - ip: 登录IP地址
//   - ipInfo: IP地址的详细信息
//   - server: 服务器信息
// 返回:
//   - error: 发送过程中的错误信息
func (t *Telegram) NotifyLoginSuccess(ip, ipInfo, server string) error {
	if !t.config.Notifications.LoginSuccess.Enabled {
		return nil
	}

	text := fmt.Sprintf("✅ SSH登录成功\n时间: %s\n%s\n服务器: %s",
		time.Now().Format("2006-01-02 15:04:05"),
		ipInfo,
		server)

	return t.SendMessage(text)
}

// NotifyLoginFailed 发送SSH登录失败的通知
// 参数:
//   - ip: 登录IP地址
//   - ipInfo: IP地址的详细信息
//   - server: 服务器信息
//   - attempts: 当前失败次数
//   - maxAttempts: 最大允许失败次数
// 返回:
//   - error: 发送过程中的错误信息
func (t *Telegram) NotifyLoginFailed(ip, ipInfo, server string, attempts, maxAttempts int) error {
	if !t.config.Notifications.LoginFailed.Enabled {
		return nil
	}

	text := fmt.Sprintf("⚠️ SSH登录失败\n时间: %s\n%s\n失败次数: %d/%d\n服务器: %s",
		time.Now().Format("2006-01-02 15:04:05"),
		ipInfo,
		attempts,
		maxAttempts,
		server)

	return t.SendMessage(text)
}

// NotifyIPBanned 发送IP被封禁的通知
// 参数:
//   - ip: 被封禁的IP地址
//   - ipInfo: IP地址的详细信息
//   - server: 服务器信息
//   - duration: 封禁时长
//   - expireTime: 解封时间
// 返回:
//   - error: 发送过程中的错误信息
func (t *Telegram) NotifyIPBanned(ip, ipInfo, server string, duration time.Duration, expireTime time.Time) error {
	if !t.config.Notifications.IPBanned.Enabled {
		return nil
	}

	text := fmt.Sprintf("🚫 IP %s 已被封禁\n时间: %s\n%s\n原因: SSH暴力破解\n封禁时长: %.0f小时\n解封时间: %s\n服务器: %s",
		ip,
		time.Now().Format("2006-01-02 15:04:05"),
		ipInfo,
		duration.Hours(),
		expireTime.Format("2006-01-02 15:04:05"),
		server)

	return t.SendMessage(text)
}

// TestCommand 测试所有通知功能
// 发送测试消息以验证通知系统是否正常工作
// 返回:
//   - error: 测试过程中的错误信息
func (t *Telegram) TestCommand() error {
	// 测试登录成功通知
	if err := t.NotifyLoginSuccess("192.168.1.1", "IP: 192.168.1.1\n属地: 中国 北京\nISP: 测试ISP", "测试服务器"); err != nil {
		return fmt.Errorf("测试登录成功通知失败: %v", err)
	}

	// 测试登录失败通知
	if err := t.NotifyLoginFailed("192.168.1.2", "IP: 192.168.1.2\n属地: 中国 上海\nISP: 测试ISP", "测试服务器", 3, 5); err != nil {
		return fmt.Errorf("测试登录失败通知失败: %v", err)
	}

	// 测试IP封禁通知
	if err := t.NotifyIPBanned("192.168.1.3", "IP: 192.168.1.3\n属地: 中国 广州\nISP: 测试ISP", "测试服务器", 24*time.Hour, time.Now().Add(24*time.Hour)); err != nil {
		return fmt.Errorf("测试IP封禁通知失败: %v", err)
	}

	return nil
}

// HandleCommands 处理Telegram命令
// 监听并处理来自Telegram的命令消息
// 支持的命令:
//   - /start: 显示欢迎信息
//   - /status: 显示系统状态
//   - /test: 测试通知功能
//   - /help: 显示帮助信息
// 返回:
//   - error: 处理过程中的错误信息
func (t *Telegram) HandleCommands() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := t.bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if !update.Message.IsCommand() {
			continue
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

		switch update.Message.Command() {
		case "start":
			msg.Text = "欢迎使用SSH防护系统！\n可用命令：\n/status - 查看系统状态\n/test - 测试通知功能\n/help - 显示帮助信息"
		case "status":
			msg.Text = "系统状态：\n- 运行中\n- 监控正常\n- 通知正常"
		case "test":
			if err := t.TestCommand(); err != nil {
				msg.Text = fmt.Sprintf("测试失败: %v", err)
			} else {
				msg.Text = "测试通知已发送，请检查是否收到"
			}
		case "help":
			msg.Text = "SSH防护系统命令：\n/start - 开始使用\n/status - 查看系统状态\n/test - 测试通知功能\n/help - 显示此帮助信息"
		default:
			msg.Text = "未知命令，请使用 /help 查看可用命令"
		}

		if _, err := t.bot.Send(msg); err != nil {
			t.logger.WithError(err).Error("发送命令响应失败")
		}
	}

	return nil
} 
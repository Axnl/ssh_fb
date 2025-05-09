package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
	"github.com/Axnl/ssh_fb/internal/config"
	"github.com/Axnl/ssh_fb/internal/monitor"
	"github.com/Axnl/ssh_fb/internal/notification"
	"github.com/Axnl/ssh_fb/pkg/firewall"
)

// 版本信息
var (
	Version   string = "unknown"
	BuildTime string = "unknown"
)

var (
	cmdInstall   bool
	cmdUninstall bool
	cmdHelp      bool
	cmdVersion   bool
)

func init() {
	flag.BoolVar(&cmdInstall, "install", false, "安装服务")
	flag.BoolVar(&cmdUninstall, "uninstall", false, "卸载服务")
	flag.BoolVar(&cmdHelp, "help", false, "显示帮助信息")
	flag.BoolVar(&cmdVersion, "version", false, "显示版本信息")
	
	flag.Usage = func() {
		fmt.Println("SSH防护系统使用说明：")
		fmt.Println("\n命令选项：")
		fmt.Println("  install  安装系统服务")
		fmt.Println("  uninstall 卸载系统服务")
		fmt.Println("  help     显示帮助信息")
		fmt.Println("  version  显示版本信息")
		fmt.Println("\n无参数启动：直接运行SSH防护系统")
		fmt.Println("\n示例：")
		fmt.Println("  ./ssh_fb         # 启动SSH防护系统")
		fmt.Println("  ./ssh_fb install # 安装服务")
		fmt.Println("  ./ssh_fb uninstall # 卸载服务")
		fmt.Println("  ./ssh_fb version  # 显示版本信息")
	}
}

func main() {
	flag.Parse()

	if !cmdInstall && !cmdUninstall && !cmdHelp && !cmdVersion && len(os.Args) > 1 {
		switch os.Args[1] {
		case "install":
			cmdInstall = true
		case "uninstall":
			cmdUninstall = true
		case "help":
			cmdHelp = true
		case "version":
			cmdVersion = true
		default:
			fmt.Printf("未知命令: %s\n", os.Args[1])
			flag.Usage()
			os.Exit(1)
		}
	}

	if cmdHelp {
		flag.Usage()
		os.Exit(0)
	}

	if cmdVersion {
		fmt.Printf("SSH防护系统\n版本: %s\n构建时间: %s\nGo版本: %s\n操作系统: %s/%s\n",
			Version, BuildTime, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	// 加载配置
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	logger := initLogger(cfg)

	if cmdInstall {
		if err := installService(cfg, logger); err != nil {
			logger.WithError(err).Fatal("服务安装失败")
		}
		logger.Info("服务安装成功")
		os.Exit(0)
	}

	if cmdUninstall {
		if err := uninstallService(cfg, logger); err != nil {
			logger.WithError(err).Fatal("服务卸载失败")
		}
		logger.Info("服务卸载成功")
		os.Exit(0)
	}

	// 正常启动程序
	logger.Info("正在启动SSH防护系统...")

	// 检查并安装必要的工具
	if err := checkAndInstallTools(logger); err != nil {
		logger.WithError(err).Fatal("工具检查/安装失败")
	}

	// 初始化Telegram通知
	telegram, err := notification.NewTelegram(&notification.Config{
		BotToken: cfg.Telegram.BotToken,
		ChatID:   cfg.Telegram.ChatID,
		Debug:    cfg.Debug.Enabled,
	}, logger)
	if err != nil {
		logger.WithError(err).Fatal("初始化Telegram通知失败")
	}

	// 启动Telegram命令处理
	go func() {
		if err := telegram.HandleCommands(); err != nil {
			logger.WithError(err).Error("Telegram命令处理失败")
		}
	}()

	// 创建并启动监控器
	mon := monitor.NewMonitor(cfg, logger, telegram)
	if err := mon.Start(); err != nil {
		logger.WithError(err).Fatal("启动监控器失败")
	}
}

func initLogger(cfg *config.Config) *logrus.Logger {
	logger := logrus.New()

	// 创建日志目录
	logDir := filepath.Dir(cfg.Logging.LogFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		logger.Fatalf("创建日志目录失败: %v", err)
	}

	// 配置日志格式
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Debug.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// 输出到文件
	file, err := os.OpenFile(cfg.Logging.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logger.Fatalf("无法创建日志文件: %v", err)
	}
	logger.SetOutput(file)

	return logger
}

func checkAndInstallTools(logger *logrus.Logger) error {
	ufw := firewall.NewUFW()
	if !ufw.IsEnabled() {
		logger.Info("正在安装ufw...")
		if err := ufw.Install(); err != nil {
			return err
		}
		logger.Info("正在启用ufw...")
		if err := ufw.Enable(); err != nil {
			return err
		}
	}
	return nil
}

func installService(cfg *config.Config, logger *logrus.Logger) error {
	logger.Info("开始安装服务")

	// 创建安装目录
	if err := os.MkdirAll(cfg.Service.InstallPath, 0755); err != nil {
		return fmt.Errorf("创建安装目录失败: %v", err)
	}

	// 复制程序文件
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取程序路径失败: %v", err)
	}

	// 复制程序
	if err := copyFile(execPath, filepath.Join(cfg.Service.InstallPath, "ssh_fb")); err != nil {
		return fmt.Errorf("复制程序失败: %v", err)
	}

	// 复制配置文件
	if err := copyFile("configs/config.yaml", filepath.Join(cfg.Service.InstallPath, "config.yaml")); err != nil {
		return fmt.Errorf("复制配置文件失败: %v", err)
	}

	// 创建服务文件
	serviceContent := fmt.Sprintf(`[Unit]
Description=SSH Protection Service
After=network.target

[Service]
Type=simple
User=%s
WorkingDirectory=%s
ExecStart=%s/ssh_fb
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target`,
		cfg.Service.User,
		cfg.Service.WorkingDirectory,
		cfg.Service.InstallPath)

	servicePath := filepath.Join("/etc/systemd/system", cfg.Service.ServiceFile)
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("创建服务文件失败: %v", err)
	}

	// 重新加载systemd配置
	cmd := exec.Command("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("重新加载systemd配置失败: %v", err)
	}

	// 启用并启动服务
	cmd = exec.Command("systemctl", "enable", cfg.Service.ServiceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("启用服务失败: %v", err)
	}

	cmd = exec.Command("systemctl", "start", cfg.Service.ServiceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("启动服务失败: %v", err)
	}

	logger.Info("服务安装完成")
	return nil
}

func uninstallService(cfg *config.Config, logger *logrus.Logger) error {
	logger.Info("开始卸载服务")

	// 停止服务
	cmd := exec.Command("systemctl", "stop", cfg.Service.ServiceName)
	if err := cmd.Run(); err != nil {
		logger.Warnf("停止服务失败: %v", err)
	}

	// 禁用服务
	cmd = exec.Command("systemctl", "disable", cfg.Service.ServiceName)
	if err := cmd.Run(); err != nil {
		logger.Warnf("禁用服务失败: %v", err)
	}

	// 删除服务文件
	servicePath := filepath.Join("/etc/systemd/system", cfg.Service.ServiceFile)
	if err := os.Remove(servicePath); err != nil {
		logger.Warnf("删除服务文件失败: %v", err)
	}

	// 重新加载systemd配置
	cmd = exec.Command("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		logger.Warnf("重新加载systemd配置失败: %v", err)
	}

	// 删除安装目录
	if err := os.RemoveAll(cfg.Service.InstallPath); err != nil {
		logger.Warnf("删除安装目录失败: %v", err)
	}

	logger.Info("服务卸载完成")
	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
} 
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Telegram struct {
		BotToken string `yaml:"bot_token"`
		ChatID   int64  `yaml:"chat_id"`
	} `yaml:"telegram"`

	SSHProtection struct {
		MaxFailedAttempts int    `yaml:"max_failed_attempts"`
		BanDurationHours  int    `yaml:"ban_duration_hours"`
		SSHLogFile        string `yaml:"ssh_log_file"`
	} `yaml:"ssh_protection"`

	Blacklist struct {
		File              string `yaml:"file"`
		CleanupIntervalHours int `yaml:"cleanup_interval_hours"`
	} `yaml:"blacklist"`

	Logging struct {
		LogFile         string `yaml:"log_file"`
		MaxSize         int    `yaml:"max_size"`
		MaxBackups      int    `yaml:"max_backups"`
		MaxAge          int    `yaml:"max_age"`
		Compress        bool   `yaml:"compress"`
		RotateInterval  int    `yaml:"rotate_interval"`
	} `yaml:"logging"`

	Service struct {
		InstallPath      string `yaml:"install_path"`
		ServiceName      string `yaml:"service_name"`
		ServiceFile      string `yaml:"service_file"`
		User             string `yaml:"user"`
		WorkingDirectory string `yaml:"working_directory"`
	} `yaml:"service"`

	IPInfo struct {
		APIURL        string `yaml:"api_url"`
		Language      string `yaml:"language"`
		Timeout       int    `yaml:"timeout"`
		RetryCount    int    `yaml:"retry_count"`
		RetryInterval int    `yaml:"retry_interval"`
	} `yaml:"ip_info"`

	Notifications struct {
		LoginSuccess struct {
			Enabled  bool   `yaml:"enabled"`
			Template string `yaml:"template"`
		} `yaml:"login_success"`
		LoginFailed struct {
			Enabled  bool   `yaml:"enabled"`
			Template string `yaml:"template"`
		} `yaml:"login_failed"`
		IPBanned struct {
			Enabled  bool   `yaml:"enabled"`
			Template string `yaml:"template"`
		} `yaml:"ip_banned"`
	} `yaml:"notifications"`

	Debug struct {
		Enabled       bool   `yaml:"enabled"`
		LogLevel      string `yaml:"log_level"`
		TraceRequests bool   `yaml:"trace_requests"`
		ProfileCPU    bool   `yaml:"profile_cpu"`
		ProfileMemory bool   `yaml:"profile_memory"`
	} `yaml:"debug"`
}

func LoadConfig(configPath string) (*Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("无法打开配置文件: %v", err)
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func validateConfig(config *Config) error {
	if config.Telegram.BotToken == "" || config.Telegram.BotToken == "your_bot_token" {
		return fmt.Errorf("Telegram配置错误: bot_token不能为空或默认值")
	}
	if config.Telegram.ChatID == 0 || config.Telegram.ChatID == 123456789 {
		return fmt.Errorf("Telegram配置错误: chat_id不能为0或默认值")
	}

	if config.SSHProtection.MaxFailedAttempts <= 0 {
		return fmt.Errorf("SSH防护配置错误: max_failed_attempts必须大于0")
	}
	if config.SSHProtection.BanDurationHours <= 0 {
		return fmt.Errorf("SSH防护配置错误: ban_duration_hours必须大于0")
	}
	if config.SSHProtection.SSHLogFile == "" {
		return fmt.Errorf("SSH防护配置错误: ssh_log_file不能为空")
	}

	if config.Blacklist.File == "" {
		return fmt.Errorf("黑名单配置错误: file不能为空")
	}
	if config.Blacklist.CleanupIntervalHours <= 0 {
		return fmt.Errorf("黑名单配置错误: cleanup_interval_hours必须大于0")
	}

	if config.Logging.LogFile == "" {
		return fmt.Errorf("日志配置错误: log_file不能为空")
	}

	if config.Service.InstallPath == "" {
		return fmt.Errorf("服务配置错误: install_path不能为空")
	}
	if config.Service.ServiceName == "" {
		return fmt.Errorf("服务配置错误: service_name不能为空")
	}

	if config.IPInfo.APIURL == "" {
		return fmt.Errorf("IP信息查询配置错误: api_url不能为空")
	}

	if runtime.GOOS == "windows" {
		if err := validateWindowsConfig(config); err != nil {
			return err
		}
	}

	return nil
}

func validateWindowsConfig(config *Config) error {
	if strings.HasPrefix(config.SSHProtection.SSHLogFile, "/var/") {
		return fmt.Errorf("SSH日志文件路径使用了Linux格式，在Windows环境下可能无法访问")
	}

	logDir := filepath.Dir(config.Logging.LogFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	blacklistDir := filepath.Dir(config.Blacklist.File)
	if err := os.MkdirAll(blacklistDir, 0755); err != nil {
		return fmt.Errorf("黑名单目录创建失败: %v", err)
	}

	return nil
} 
// Package firewall 提供防火墙管理功能
package firewall

import (
	"fmt"
	"os/exec"
)

// UFW 结构体封装了UFW防火墙的操作
type UFW struct{}

// NewUFW 创建并初始化一个新的UFW防火墙管理器
// 返回:
//   - *UFW: 初始化后的UFW实例
func NewUFW() *UFW {
	return &UFW{}
}

// BanIP 封禁指定的IP地址
// 参数:
//   - ip: 要封禁的IP地址
// 返回:
//   - error: 封禁过程中的错误信息
func (u *UFW) BanIP(ip string) error {
	cmd := exec.Command("ufw", "deny", "from", ip, "to", "any")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("封禁IP失败 %s: %v", ip, err)
	}
	return nil
}

// UnbanIP 解除指定IP地址的封禁
// 参数:
//   - ip: 要解除封禁的IP地址
// 返回:
//   - error: 解除封禁过程中的错误信息
func (u *UFW) UnbanIP(ip string) error {
	cmd := exec.Command("ufw", "delete", "deny", "from", ip, "to", "any")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("解除IP封禁失败 %s: %v", ip, err)
	}
	return nil
}

// IsEnabled 检查UFW防火墙是否已启用
// 返回:
//   - bool: true表示已启用，false表示未启用
func (u *UFW) IsEnabled() bool {
	cmd := exec.Command("ufw", "status")
	return cmd.Run() == nil
}

// Enable 启用UFW防火墙
// 返回:
//   - error: 启用过程中的错误信息
func (u *UFW) Enable() error {
	cmd := exec.Command("ufw", "enable")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("启用ufw失败: %v", err)
	}
	return nil
}

// Install 安装UFW防火墙
// 支持以下包管理器:
//   - apt-get (Debian/Ubuntu)
//   - yum (CentOS/RHEL)
//   - dnf (Fedora)
// 返回:
//   - error: 安装过程中的错误信息
func (u *UFW) Install() error {
	// 检查包管理器
	var installCmd *exec.Cmd
	if _, err := exec.LookPath("apt-get"); err == nil {
		installCmd = exec.Command("apt-get", "update")
		if err := installCmd.Run(); err != nil {
			return fmt.Errorf("更新包列表失败: %v", err)
		}
		installCmd = exec.Command("apt-get", "install", "-y", "ufw")
	} else if _, err := exec.LookPath("yum"); err == nil {
		installCmd = exec.Command("yum", "install", "-y", "ufw")
	} else if _, err := exec.LookPath("dnf"); err == nil {
		installCmd = exec.Command("dnf", "install", "-y", "ufw")
	} else {
		return fmt.Errorf("未找到支持的包管理器")
	}

	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("安装ufw失败: %v", err)
	}

	return nil
} 
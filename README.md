# SSH防护系统

一个用于监控和防护SSH暴力破解的工具，支持自动封禁IP、Telegram通知等功能。

## 功能特点

- 监控SSH登录失败尝试
- 自动封禁暴力破解IP
- Telegram实时通知
- IP地理位置查询
- 支持Windows和Linux系统
- 系统服务安装/卸载
- 日志轮转
- 黑名单管理
- Telegram命令支持

## 系统要求

- Go 1.21或更高版本
- Linux系统（推荐）或Windows系统
- UFW防火墙（Linux）
- Telegram Bot Token

## 安装

1. 克隆仓库：
```bash
git clone https://github.com/yourusername/ssh_fb.git
cd ssh_fb
```

2. 编译：
```bash
# 使用自动编译脚本（推荐）
chmod +x scripts/build.sh
./scripts/build.sh

# 或手动编译
go build -o ssh_fb cmd/ssh_fb/main.go
```

3. 配置：
编辑 `configs/config.yaml` 文件，设置：
- Telegram Bot Token
- Chat ID
- 其他配置项

4. 安装服务：
```bash
sudo ./ssh_fb install
```

## 使用方法

1. 启动服务：
```bash
sudo systemctl start ssh_fb
```

2. 查看状态：
```bash
sudo systemctl status ssh_fb
```

3. 查看日志：
```bash
sudo journalctl -u ssh_fb -f
```

4. 卸载服务：
```bash
sudo ./ssh_fb uninstall
```

5. 查看版本信息：
```bash
./ssh_fb version
```

## Telegram命令

系统支持以下Telegram命令：

- `/start` - 开始使用，显示欢迎信息
- `/status` - 查看系统状态
- `/test` - 测试通知功能
- `/help` - 显示帮助信息

## 配置说明

配置文件 `configs/config.yaml` 包含以下主要配置项：

- Telegram配置
- SSH防护配置
- 黑名单配置
- 日志配置
- 服务配置
- IP信息查询配置
- 通知消息模板
- 调试配置

## 开发

1. 安装依赖：
```bash
go mod download
```

2. 运行测试：
```bash
go test ./...
```

3. 构建：
```bash
# 使用自动编译脚本（推荐）
./scripts/build.sh

# 或手动编译
go build -o ssh_fb cmd/ssh_fb/main.go
```

## 自动编译脚本说明

`scripts/build.sh` 脚本提供以下功能：

1. 检查系统依赖：
   - Go 1.21或更高版本
   - git
   - make

2. 自动安装缺失的系统依赖：
   - 支持 Debian/Ubuntu (apt-get)
   - 支持 CentOS/RHEL (yum)

3. 自动下载Go依赖：
   - 使用 `go mod download`

4. 编译项目：
   - 自动设置版本信息
   - 自动设置构建时间
   - 生成可执行文件

5. 复制必要文件：
   - 配置文件
   - 脚本文件

输出目录：`build/`

## 许可证

MIT License 
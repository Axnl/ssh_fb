#!/bin/bash

# 设置颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印带颜色的信息
info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查命令是否存在
check_command() {
    if ! command -v $1 &> /dev/null; then
        return 1
    fi
    return 0
}

# 检查并创建目录
check_and_create_dir() {
    local dir=$1
    if [ ! -d "$dir" ]; then
        warn "目录 $dir 不存在，正在创建..."
        mkdir -p "$dir"
        if [ $? -ne 0 ]; then
            error "创建目录 $dir 失败"
            return 1
        fi
        info "目录 $dir 创建成功"
    fi
    return 0
}

# 检查并创建文件
check_and_create_file() {
    local file=$1
    local content=$2
    if [ ! -f "$file" ]; then
        warn "文件 $file 不存在，正在创建..."
        echo "$content" > "$file"
        if [ $? -ne 0 ]; then
            error "创建文件 $file 失败"
            return 1
        fi
        info "文件 $file 创建成功"
    fi
    return 0
}

# 检查项目结构
check_project_structure() {
    info "检查项目结构..."

    # 检查必要的目录
    local dirs=(
        "cmd/ssh_fb"
        "internal/config"
        "internal/service"
        "internal/monitor"
        "internal/notification"
        "pkg/ipinfo"
        "pkg/firewall"
        "configs"
        "scripts"
    )

    for dir in "${dirs[@]}"; do
        if ! check_and_create_dir "$dir"; then
            return 1
        fi
    done

    # 检查必要的文件
    if [ ! -f "go.mod" ]; then
        warn "go.mod 文件不存在，正在创建..."
        go mod init github.com/yourusername/ssh_fb
        if [ $? -ne 0 ]; then
            error "创建 go.mod 文件失败"
            return 1
        fi
    fi

    # 检查配置文件
    if [ ! -f "configs/config.yaml" ]; then
        warn "配置文件不存在，正在创建..."
        check_and_create_file "configs/config.yaml" "# SSH防护系统配置文件
telegram:
  bot_token: \"your_bot_token\"
  chat_id: \"your_chat_id\"

ssh_protection:
  max_failed_attempts: 5
  ban_duration_hours: 24
  ssh_log_file: \"/var/log/auth.log\"

blacklist:
  file: \"/var/lib/ssh_fb/blacklist.txt\"
  cleanup_interval_hours: 24

logging:
  log_file: \"/var/log/ssh_fb/ssh_fb.log\"
  max_size: 100
  max_backups: 3
  max_age: 28
  compress: true
  rotate_interval: 24

service:
  install_path: \"/usr/local/ssh_fb\"
  service_name: \"ssh_fb\"
  service_file: \"ssh_fb.service\"
  user: \"root\"
  working_directory: \"/usr/local/ssh_fb\"

ipinfo:
  api_url: \"https://ipapi.co\"
  language: \"zh\"
  timeout: 5
  retry_count: 3
  retry_interval: 1

notifications:
  login_success_template: \"✅ 登录成功\n时间: {{.Time}}\nIP: {{.IP}}\n位置: {{.Location}}\n服务器: {{.Server}}\"
  login_failed_template: \"❌ 登录失败\n时间: {{.Time}}\nIP: {{.IP}}\n尝试次数: {{.Attempts}}\n服务器: {{.Server}}\"
  ip_banned_template: \"🚫 IP已封禁\nIP: {{.IP}}\n封禁时长: {{.Duration}}小时\n到期时间: {{.ExpireTime}}\n服务器: {{.Server}}\"

debug:
  enabled: false
  log_level: \"info\""
    fi

    info "项目结构检查完成"
    return 0
}

# 设置Go代理
setup_goproxy() {
    info "检查Go代理设置..."

    # 检查是否已设置GOPROXY
    if [ -z "$GOPROXY" ]; then
        warn "GOPROXY未设置，正在设置阿里云代理..."
        go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
        if [ $? -ne 0 ]; then
            error "设置GOPROXY失败"
            return 1
        fi
        info "GOPROXY设置成功"
    elif [[ "$GOPROXY" != *"mirrors.aliyun.com"* ]]; then
        warn "GOPROXY不是阿里云代理，正在更新..."
        go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
        if [ $? -ne 0 ]; then
            error "更新GOPROXY失败"
            return 1
        fi
        info "GOPROXY更新成功"
    else
        info "GOPROXY已正确设置"
    fi

    return 0
}

# 检查并安装依赖
install_dependencies() {
    info "检查系统依赖..."
    
    # 运行安装脚本
    if [ -f "scripts/install.sh" ]; then
        chmod +x scripts/install.sh
        ./scripts/install.sh
        if [ $? -ne 0 ]; then
            error "安装依赖失败"
            exit 1
        fi
    else
        error "install.sh 脚本不存在"
        exit 1
    fi

    # 检查并安装Go依赖
    info "检查Go依赖..."
    if [ -f "go.mod" ]; then
        go mod download
        if [ $? -ne 0 ]; then
            error "下载Go依赖失败"
            exit 1
        fi
    else
        error "go.mod文件不存在"
        exit 1
    fi
}

# 清理旧的构建文件
cleanup() {
    info "清理旧的构建文件..."
    rm -rf build/
    mkdir -p build
}

# 编译项目
build() {
    info "开始编译项目..."
    
    # 设置编译参数
    VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "unknown")
    BUILD_TIME=$(date +%FT%T%z)
    GOOS=$(go env GOOS)
    GOARCH=$(go env GOARCH)

    # 编译
    go build -v -o build/ssh_fb \
        -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" \
        ./cmd/ssh_fb

    if [ $? -ne 0 ]; then
        error "编译失败"
        exit 1
    fi

    # 复制配置文件
    if [ -d "configs" ]; then
        cp -r configs build/
    fi

    # 复制脚本文件
    if [ -d "scripts" ]; then
        cp -r scripts build/
    fi

    info "编译完成，输出目录: build/"
}

# 主函数
main() {
    info "开始构建SSH防护系统..."
    
    # 检查项目结构
    if ! check_project_structure; then
        error "项目结构检查失败"
        exit 1
    fi

    # 设置Go代理
    if ! setup_goproxy; then
        error "设置Go代理失败"
        exit 1
    fi
    
    # 检查并安装依赖
    install_dependencies
    
    # 清理旧的构建文件
    cleanup
    
    # 编译项目
    build
    
    info "构建完成！"
    info "可执行文件位置: build/ssh_fb"
}

# 执行主函数
main 
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

# 检测操作系统类型
detect_os() {
    if [ -f /etc/debian_version ]; then
        echo "debian"
    elif [ -f /etc/redhat-release ]; then
        echo "redhat"
    elif [ -f /etc/arch-release ]; then
        echo "arch"
    elif [ -f /etc/alpine-release ]; then
        echo "alpine"
    else
        echo "unknown"
    fi
}

# 获取系统架构
get_arch() {
    local arch=$(uname -m)
    case $arch in
        "x86_64")
            echo "amd64"
            ;;
        "aarch64")
            echo "arm64"
            ;;
        "armv7l")
            echo "armv6l"
            ;;
        *)
            echo $arch
            ;;
    esac
}

# 安装包管理器
install_package_manager() {
    local os_type=$1
    case $os_type in
        "debian")
            if ! check_command "apt-get"; then
                warn "正在安装 apt-get..."
                sudo apt-get update
            fi
            ;;
        "redhat")
            if ! check_command "yum"; then
                warn "正在安装 yum..."
                sudo yum install -y yum-utils
            fi
            ;;
        "arch")
            if ! check_command "pacman"; then
                warn "正在安装 pacman..."
                sudo pacman -Syu --noconfirm
            fi
            ;;
        "alpine")
            if ! check_command "apk"; then
                warn "正在安装 apk..."
                sudo apk update
            fi
            ;;
    esac
}

# 安装工具
install_tool() {
    local tool=$1
    local os_type=$2

    if check_command $tool; then
        info "$tool 已安装"
        return 0
    fi

    warn "正在安装 $tool..."
    case $os_type in
        "debian")
            sudo apt-get install -y $tool
            ;;
        "redhat")
            sudo yum install -y $tool
            ;;
        "arch")
            sudo pacman -S --noconfirm $tool
            ;;
        "alpine")
            sudo apk add $tool
            ;;
        *)
            error "不支持的操作系统"
            return 1
            ;;
    esac

    if [ $? -eq 0 ]; then
        info "$tool 安装成功"
        return 0
    else
        error "$tool 安装失败"
        return 1
    fi
}

# 卸载现有的Go
uninstall_go() {
    info "正在卸载现有的Go安装..."
    
    # 删除Go安装目录
    if [ -d "/usr/local/go" ]; then
        sudo rm -rf /usr/local/go
    fi

    # 从PATH中移除Go
    if grep -q "export PATH=\$PATH:/usr/local/go/bin" /etc/profile; then
        sudo sed -i '/export PATH=$PATH:\/usr\/local\/go\/bin/d' /etc/profile
    fi

    # 删除Go缓存
    if [ -d "$HOME/go" ]; then
        rm -rf "$HOME/go"
    fi

    # 删除Go可执行文件
    if [ -f "/usr/bin/go" ]; then
        sudo rm -f /usr/bin/go
    fi

    info "Go卸载完成"
}

# 获取最新的Go版本
get_latest_go_version() {
    local version=""
    
    # 尝试使用curl
    if check_command "curl"; then
        version=$(curl -s https://go.dev/VERSION?m=text)
        if [ $? -eq 0 ] && [ ! -z "$version" ]; then
            echo "$version"
            return 0
        fi
    fi
    
    # 如果curl失败，尝试使用wget
    if check_command "wget"; then
        version=$(wget -qO- https://go.dev/VERSION?m=text)
        if [ $? -eq 0 ] && [ ! -z "$version" ]; then
            echo "$version"
            return 0
        fi
    fi
    
    return 1
}

# 使用curl下载
download_with_curl() {
    local url=$1
    local output=$2
    local timeout=30
    
    curl --connect-timeout $timeout \
         --retry 3 \
         --retry-delay 1 \
         --retry-max-time 60 \
         -L \
         -o "$output" \
         "$url"
    
    return $?
}

# 使用wget下载
download_with_wget() {
    local url=$1
    local output=$2
    local timeout=30
    
    wget --timeout=$timeout \
         --tries=3 \
         --waitretry=1 \
         --no-check-certificate \
         -q \
         -O "$output" \
         "$url"
    
    return $?
}

# 下载文件
download_file() {
    local url=$1
    local output=$2
    local retries=3
    local success=0

    for ((i=1; i<=retries; i++)); do
        # 首先尝试使用curl
        if check_command "curl"; then
            info "尝试使用curl下载..."
            if download_with_curl "$url" "$output"; then
                success=1
                break
            fi
        fi

        # 如果curl失败或不可用，尝试使用wget
        if check_command "wget"; then
            info "尝试使用wget下载..."
            if download_with_wget "$url" "$output"; then
                success=1
                break
            fi
        fi

        if [ $i -lt $retries ]; then
            warn "第 $i 次下载失败，正在重试..."
            sleep 2
        fi
    done

    if [ $success -eq 1 ]; then
        return 0
    else
        error "所有下载尝试均失败"
        return 1
    fi
}

# 从官网下载Go
download_go() {
    local version=$1
    local arch=$2
    local os_name=$3
    local output=$4
    local filename="go${version}.${os_name}-${arch}.tar.gz"
    local download_url="https://go.dev/dl/${filename}"

    info "正在从官网下载 Go ${version}..."
    if ! download_file "$download_url" "$output"; then
        error "下载失败"
        return 1
    fi

    info "下载成功"
    return 0
}

# 安装Go
install_go() {
    local os_type=$1
    local required_version="1.21.0"
    local current_version=""

    # 检查是否已安装Go
    if check_command "go"; then
        current_version=$(go version | awk '{print $3}' | sed 's/go//')
        if [ $(echo "$current_version $required_version" | awk '{print ($1 >= $2)}') -eq 1 ]; then
            info "Go $current_version 已安装，版本满足要求"
            return 0
        else
            warn "Go版本过低（当前版本：$current_version，需要版本：$required_version）"
            uninstall_go
        fi
    fi

    # 获取最新版本
    local latest_version=$(get_latest_go_version)
    if [ $? -ne 0 ] || [ -z "$latest_version" ]; then
        error "获取最新版本失败"
        return 1
    fi
    info "获取到最新版本：$latest_version"

    # 创建临时目录
    local temp_dir=$(mktemp -d)
    cd $temp_dir

    # 下载并安装Go
    local arch=$(get_arch)
    local os_name="linux"
    local download_file="go${latest_version}.${os_name}-${arch}.tar.gz"

    if ! download_go "$latest_version" "$arch" "$os_name" "$download_file"; then
        cd - > /dev/null
        rm -rf $temp_dir
        return 1
    fi

    # 解压安装包
    info "正在解压安装包..."
    if ! sudo tar -C /usr/local -xzf "$download_file"; then
        error "解压Go安装包失败"
        cd - > /dev/null
        rm -rf $temp_dir
        return 1
    fi

    # 设置环境变量
    if ! grep -q "export PATH=\$PATH:/usr/local/go/bin" /etc/profile; then
        echo "export PATH=\$PATH:/usr/local/go/bin" | sudo tee -a /etc/profile
    fi
    source /etc/profile

    # 清理临时文件
    cd - > /dev/null
    rm -rf $temp_dir

    # 验证安装
    if ! command -v go &> /dev/null; then
        error "Go安装失败"
        return 1
    fi

    local installed_version=$(go version | awk '{print $3}' | sed 's/go//')
    if [ "$installed_version" = "$latest_version" ]; then
        info "Go ${installed_version} 安装成功"
        return 0
    else
        error "Go版本不匹配，安装失败"
        return 1
    fi
}

# 主函数
main() {
    info "开始检查系统依赖..."

    # 检测操作系统
    local os_type=$(detect_os)
    if [ "$os_type" = "unknown" ]; then
        error "无法检测操作系统类型"
        exit 1
    fi
    info "检测到操作系统类型: $os_type"

    # 安装包管理器
    install_package_manager $os_type

    # 安装基本工具
    local tools=("git" "make" "gcc" "wget" "curl")
    for tool in "${tools[@]}"; do
        install_tool $tool $os_type
    done

    # 安装Go
    install_go $os_type

    info "系统依赖检查完成"
}

# 执行主函数
main 
#!/bin/bash

# ImageSpider 编译脚本
# 使用方法: ./build.sh [frontend|backend|all|clean]

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo_error() {
    echo -e "${RED}[错误]${NC} $1"
}

echo_success() {
    echo -e "${GREEN}[成功]${NC} $1"
}

echo_info() {
    echo -e "${YELLOW}[信息]${NC} $1"
}

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 编译前端
build_frontend() {
    echo_info "开始编译前端..."

    if [ ! -d "frontend" ]; then
        echo_error "frontend 目录不存在"
        exit 1
    fi

    cd frontend

    # 检查 node_modules 是否存在
    if [ ! -d "node_modules" ]; then
        echo_info "安装前端依赖..."
        npm install
    fi

    # 编译前端
    echo_info "编译前端代码..."
    npm run build

    cd ..
    echo_success "前端编译完成"
}

# 编译后端
build_backend() {
    echo_info "开始编译后端..."

    # 检查 go.mod 是否存在
    if [ ! -f "go.mod" ]; then
        echo_error "go.mod 不存在，这不是一个 Go 项目目录"
        exit 1
    fi

    # 编译 Go 程序
    echo_info "编译 Go 程序..."
    go build -o imagespider .

    echo_success "后端编译完成，生成文件: imagespider"
}

# 清理编译产物
clean() {
    echo_info "清理编译产物..."

    # 删除前端编译产物
    if [ -d "embed/www" ]; then
        rm -rf embed/www
        echo_info "已删除 embed/www"
    fi

    # 删除后端编译产物
    if [ -f "imagespider" ]; then
        rm -f imagespider
        echo_info "已删除 imagespider"
    fi

    # 删除前端依赖
    if [ -d "frontend/node_modules" ]; then
        read -p "是否删除 frontend/node_modules? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            rm -rf frontend/node_modules
            echo_info "已删除 frontend/node_modules"
        fi
    fi

    echo_success "清理完成"
}

# 显示使用说明
usage() {
    cat << EOF
ImageSpider 编译脚本

使用方法: $0 [frontend|backend|all|clean]

参数说明:
    frontend    只编译前端
    backend     只编译后端
    all         编译前端和后端 (默认)
    clean       清理编译产物
    help        显示帮助信息

示例:
    $0          # 编译前端和后端
    $0 frontend # 只编译前端
    $0 backend  # 只编译后端
    $0 clean    # 清理编译产物

EOF
}

# 主逻辑
case "${1:-all}" in
    frontend)
        build_frontend
        ;;
    backend)
        build_backend
        ;;
    all)
        build_frontend
        build_backend
        ;;
    clean)
        clean
        ;;
    help|--help|-h)
        usage
        ;;
    *)
        echo_error "未知参数: $1"
        usage
        exit 1
        ;;
esac

echo_success "编译完成!"

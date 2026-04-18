#!/bin/bash
# 遇到错误立即退出
set -e

# --- 配置区 ---
REPO="kiritosuki/qsh"
BINARY_NAME="qsh"

# 支持 VERSION=latest 或 VERSION=v0.1.0 传入
if [ -z "$VERSION" ] || [ "$VERSION" == "latest" ]; then
    echo "------------------------------------------------"
    echo "Fetching the latest version info..."
    # 增加 -f 标志，如果 API 挂了直接报错
    VERSION=$(curl -sf "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
fi

# --- 架构检测 ---
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Error: Architecture $ARCH is not supported."; exit 1 ;;
esac

if [[ "$OS" != "linux" && "$OS" != "darwin" ]]; then
    echo "Error: This script only supports Linux and macOS."
    exit 1
fi

# 匹配产物格式
RELEASE_TAR="${BINARY_NAME}-${OS}-${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${RELEASE_TAR}"

# --- 执行下载与安装 ---
echo "------------------------------------------------"
echo "Start downloading ${BINARY_NAME} ${VERSION}..."
echo "Platform: ${OS}/${ARCH}"
echo "------------------------------------------------"

# 1. 创建临时目录
TMP_DIR=$(mktemp -d)
# 确保脚本退出时清理临时目录
trap 'rm -rf "$TMP_DIR"' EXIT

# 2. 下载
curl -L "$DOWNLOAD_URL" -o "${TMP_DIR}/${RELEASE_TAR}"

# 3. 解压
echo "Extracting files..."
tar -xzf "${TMP_DIR}/${RELEASE_TAR}" -C "$TMP_DIR"

# 4. 寻找解压后的二进制文件 (改用更稳健的查找方式)
# 只要找到包含 qsh-os-arch 字样的第一个文件即可
SOURCE_BINARY=$(find "$TMP_DIR" -type f -name "${BINARY_NAME}-${OS}-${ARCH}*" | head -n 1)

if [ -z "$SOURCE_BINARY" ]; then
    echo "Error: Could not find binary in the downloaded package."
    exit 1
fi

# 5. 安装
echo "Installing to /usr/local/bin/q (may require password)..."
sudo mv "$SOURCE_BINARY" "/usr/local/bin/q"
sudo chmod +x "/usr/local/bin/q"

# --- 安装完成提示 ---
echo "------------------------------------------------"
echo "Installation successful!"
echo ""
echo "Next steps to get started:"
echo "------------------------------------------------"
echo "  1. Set your API Key (Mandatory):"
echo "     export QSH_API_KEY=YOUR_API_KEY"
echo ""
echo "  2. Get your key from:"
echo "     https://console.aihubmix.com/"
echo ""
echo "  3. Try running your first query:"
echo "     q \"how to check port 8080\""
echo "------------------------------------------------"
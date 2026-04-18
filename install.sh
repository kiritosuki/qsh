#!/bin/bash
# 遇到错误立即退出
set -e

# --- 配置区 ---
REPO="kiritosuki/qsh"
BINARY_NAME="qsh"

# 支持 VERSION=latest 或通过环境变量 VERSION=v0.1.0 指定
if [ -z "$VERSION" ] || [ "$VERSION" == "latest" ]; then
    echo "------------------------------------------------"
    echo "Fetching the latest version info..."
    # 获取 GitHub 最新 Release 的 Tag 名字
    VERSION=$(curl -sf "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
fi

# --- 架构检测 (macOS & Linux) ---
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Error: Architecture $ARCH is not supported."; exit 1 ;;
esac

# 验证操作系统
if [[ "$OS" != "linux" && "$OS" != "darwin" ]]; then
    echo "Error: This script only supports Linux and macOS."
    exit 1
fi

# 匹配你 build.sh 产生的压缩包格式: qsh-darwin-arm64.tar.gz
RELEASE_TAR="${BINARY_NAME}-${OS}-${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${RELEASE_TAR}"

# --- 执行下载与安装 ---
echo "------------------------------------------------"
echo "Start downloading ${BINARY_NAME} ${VERSION}..."
echo "Platform: ${OS}/${ARCH}"
echo "------------------------------------------------"

# 1. 创建临时目录
TMP_DIR=$(mktemp -d)
# 确保脚本无论成功还是失败，退出时都会清理临时目录
trap 'rm -rf "$TMP_DIR"' EXIT

# 2. 下载压缩包
curl -L "$DOWNLOAD_URL" -o "${TMP_DIR}/${RELEASE_TAR}"

# 3. 解压文件
echo "Extracting files..."
tar -xzf "${TMP_DIR}/${RELEASE_TAR}" -C "$TMP_DIR"

# 4. 寻找解压后的二进制文件
# 核心修复：排除 .tar.gz 后缀，确保只匹配到二进制程序本身
SOURCE_BINARY=$(find "$TMP_DIR" -type f -name "${BINARY_NAME}-${OS}-${ARCH}*" ! -name "*.tar.gz" | head -n 1)

if [ -z "$SOURCE_BINARY" ]; then
    echo "Error: Could not find binary in the downloaded package."
    exit 1
fi

# 5. 安装到系统路径并重命名为 'q'
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
#!/bin/bash

# 1. 准备目录
mkdir -p dist

# 2. 核心编译逻辑：OS/ARCH 组合
# 涵盖了：Linux(amd64/arm64), Windows(amd64/arm64), macOS(amd64/arm64)
for platform in linux/amd64 linux/arm64 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64; do
    # 拆分变量
    os=${platform%/*}
    arch=${platform#*/}
    
    # 处理 Windows 的后缀名
    ext=""
    [[ "$os" == "windows" ]] && ext=".exe"

    echo "Building $os-$arch..."

    # 一行搞定编译：禁用CGO确保静态链接，LD_FLAGS瘦身
    CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -ldflags="-s -w" -o "dist/qsh-${os}-${arch}${ext}" .
done

echo "Done! Check the 'dist' folder."

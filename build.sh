#!/bin/bash

# 1. 准备并清理目录
mkdir -p dist
rm -f dist/*

# 2. 核心编译与压缩逻辑
for platform in linux/amd64 linux/arm64 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64; do
    # 拆分变量
    os=${platform%/*}
    arch=${platform#*/}

    # 处理 Windows 的后缀名
    ext=""
    [[ "$os" == "windows" ]] && ext=".exe"

    # 定义文件名格式
    # 原始二进制文件名：qsh-linux-amd64
    binary_name="qsh-${os}-${arch}${ext}"
    # 压缩包文件名：qsh-linux-amd64.tar.gz
    tar_name="qsh-${os}-${arch}.tar.gz"

    echo "Building $os-$arch..."

    # 执行编译
    CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -ldflags="-s -w" -o "dist/${binary_name}" .

    # 执行压缩
    # -C dist 表示切换到 dist 目录执行，这样压缩包里不会包含多余的路径层级
    echo "Compressing $tar_name..."
    tar -czf "dist/${tar_name}" -C dist "${binary_name}"
done

echo "---------------------------------------"
echo "Done! Final files in 'dist':"
ls -lh dist
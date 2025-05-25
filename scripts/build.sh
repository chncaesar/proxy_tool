 #!/bin/bash

set -e

# 版本信息
VERSION="0.1.0"
BUILD_TIME=$(date "+%F %T")
LDFLAGS="-X 'main.Version=${VERSION}' -X 'main.BuildTime=${BUILD_TIME}'"

# 创建输出目录
mkdir -p build

# 构建 Linux amd64 版本
echo "构建 Linux amd64 版本..."
GOOS=linux GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o build/proxy-tool-linux-amd64 cmd/main.go

# 构建 macOS 版本
echo "构建 macOS 版本..."
GOOS=darwin GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o build/proxy-tool-darwin-amd64 cmd/main.go
GOOS=darwin GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o build/proxy-tool-darwin-arm64 cmd/main.go

# 创建发布包
echo "创建发布包..."
cd build
tar czf proxy-tool-linux-amd64.tar.gz proxy-tool-linux-amd64
tar czf proxy-tool-darwin-amd64.tar.gz proxy-tool-darwin-amd64
tar czf proxy-tool-darwin-arm64.tar.gz proxy-tool-darwin-arm64

echo "构建完成！"
echo "输出文件："
echo "- Linux amd64: build/proxy-tool-linux-amd64.tar.gz"
echo "- macOS Intel: build/proxy-tool-darwin-amd64.tar.gz"
echo "- macOS ARM:   build/proxy-tool-darwin-arm64.tar.gz"
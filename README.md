# Proxy Tool

这是一个用于配置系统代理的命令行工具，支持配置 git、docker、apt 等服务的代理设置。

## 功能特点

- 支持配置多个系统服务的代理
  - Git 代理设置
  - Docker 代理设置
  - APT 代理设置
- 简单的命令行接口
- 支持查看当前代理设置

## 开发环境要求

- Go 1.21 或更高版本
- Linux 系统
- root 权限（用于修改系统配置）

## 快速开始

1. 克隆项目
```bash
git clone [项目地址]
cd proxy_tool
```

2. 安装依赖
```bash
go mod download
```

3. 构建项目
```bash
go build -o proxy-tool cmd/main.go
```

4. 使用工具
```bash
# 设置系统代理（需要 root 权限）
sudo proxy-tool set localhost:7890

# 查看当前代理设置
proxy-tool get

# 查看版本信息
proxy-tool version
```

## 命令行使用说明

```bash
proxy-tool <command> [arguments]

命令:
  set <address>    设置系统代理，地址格式为 host:port
  get             显示当前代理设置
  version         显示版本信息

示例:
  sudo proxy-tool set localhost:7890
  proxy-tool get
```

## 注意事项

1. 设置代理需要 root 权限
2. 修改 Docker 配置后会自动重启 Docker 服务
3. 建议在设置代理前备份相关配置文件

## 测试

运行测试：
```bash
go test ./...
```

## 许可证

MIT License
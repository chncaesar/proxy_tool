# Proxy Tool

这是一个用于配置系统代理的命令行工具，支持配置 git、docker、apt 等服务的代理设置。

## 功能特点

- 支持配置多个服务的代理
  - Git 代理设置（用户级）
  - Docker 代理设置
    - Docker 客户端配置（用户级，~/.docker/config.json）
    - Docker 守护进程配置（系统级，/etc/docker/daemon.json）
  - APT 代理设置（系统级）
  - 环境变量代理设置（用户级，~/.bashrc, ~/.zshrc, ~/.bash_profile）

- 支持用户级和系统级代理配置
- 简单的命令行接口
- 支持查看当前代理设置
- 自动备份配置文件

## 开发环境要求

- Go 1.21 或更高版本
- Linux 系统
- root 权限（用于修改系统级配置）

## 快速开始

1. 克隆项目
```bash
git clone https://github.com/chncaesar/proxy_tool
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
# 设置系统级代理（需要 root 权限）
sudo proxy-tool set --type=system localhost:7890

# 设置用户级代理（不需要 root 权限）
proxy-tool set --type=user localhost:7890

# 查看当前代理设置
proxy-tool get

# 查看版本信息
proxy-tool version
```

## 命令行使用说明

```bash
proxy-tool <command> [arguments]

命令:
  set [--type=system|user] <address>    设置代理，地址格式为 host:port
                                        --type 参数可选，默认为 system
  get                                  显示当前代理设置
  version                              显示版本信息

示例:
  sudo proxy-tool set --type=system localhost:7890  # 设置系统级代理
  proxy-tool set --type=user localhost:7890        # 设置用户级代理
  proxy-tool get                                   # 查看所有代理设置
```

## 代理配置说明

### 系统级代理（需要 root 权限）
- Docker 守护进程（/etc/docker/daemon.json）
  - 配置 Docker 守护进程的代理设置
  - 需要重启 Docker 服务才能生效
- APT（/etc/apt/apt.conf.d/02proxy.conf）
  - 配置 APT 包管理器的代理设置

### 用户级代理（不需要 root 权限）
- Docker 客户端（~/.docker/config.json）
  - 配置 Docker 客户端的代理设置
  - 保留现有的认证信息（auths）等配置
- Git（~/.gitconfig）
  - 配置 Git 的全局代理设置
- 环境变量（~/.bashrc, ~/.zshrc, ~/.bash_profile）
  - 配置 http_proxy, https_proxy, all_proxy 环境变量
  - 自动检测并更新用户使用的 shell 配置文件
  - 配置后需要重新加载配置文件或重启终端才能生效
  - 可以使用 `source ~/.bashrc` 或 `source ~/.zshrc` 重新加载配置  

## 注意事项

1. 设置系统级代理需要 root 权限
2. 修改 Docker 守护进程配置后需要重启 Docker 服务
3. 工具会自动备份配置文件，备份文件名为原文件名加上 .bak 后缀
4. 建议在设置代理前检查相关配置文件的内容
5. 环境变量代理配置后需要重新加载配置文件或重启终端才能生效


## 测试

运行测试：
```bash
go test ./...
```

## 许可证

MIT License
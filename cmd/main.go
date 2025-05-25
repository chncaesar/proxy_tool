package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type ProxyService struct {
	Name     string
	SetProxy func(string) error
	GetProxy func() (string, error)
}

var services = []ProxyService{
	{
		Name: "git",
		SetProxy: func(addr string) error {
			cmd := exec.Command("git", "config", "--global", "http.proxy", "http://"+addr)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("设置 git http 代理失败: %v", err)
			}
			cmd = exec.Command("git", "config", "--global", "https.proxy", "http://"+addr)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("设置 git https 代理失败: %v", err)
			}
			return nil
		},
		GetProxy: func() (string, error) {
			cmd := exec.Command("git", "config", "--global", "http.proxy")
			output, err := cmd.Output()
			if err != nil {
				return "", fmt.Errorf("获取 git 代理设置失败: %v", err)
			}
			cmd2 := exec.Command("git", "config", "--global", "https.proxy")
			output2, err := cmd2.Output()
			if err != nil {
				return "", fmt.Errorf("获取 git https代理设置失败: %v", err)
			}
			return strings.TrimSpace("http.proxy="+string(output)) + ",https.proxy=" + strings.TrimSpace(string(output2)), nil
		},
	},
	{
		Name: "docker",
		SetProxy: func(addr string) error {
			dockerConfigPath := "/etc/docker/daemon.json"
			backupPath := dockerConfigPath + ".bak"

			// 准备新的代理配置
			proxyConfig := map[string]interface{}{
				"proxies": map[string]interface{}{
					"default": map[string]interface{}{
						"httpProxy":  "http://" + addr,
						"httpsProxy": "http://" + addr,
						"noProxy":    "localhost,127.0.0.1",
					},
				},
			}

			// 读取现有配置
			var existingConfig map[string]interface{}
			if _, err := os.Stat(dockerConfigPath); err == nil {
				// 备份当前配置
				currentConfig, err := os.ReadFile(dockerConfigPath)
				if err != nil {
					return fmt.Errorf("读取 docker 配置失败: %v", err)
				}
				if err := os.WriteFile(backupPath, currentConfig, 0644); err != nil {
					return fmt.Errorf("备份 docker 配置失败: %v", err)
				}
				fmt.Printf("已备份 docker 配置文件到: %s\n", backupPath)

				// 解析现有配置
				if err := json.Unmarshal(currentConfig, &existingConfig); err != nil {
					return fmt.Errorf("解析 docker 配置失败: %v", err)
				}
			} else {
				existingConfig = make(map[string]interface{})
			}

			// 合并配置
			if existingConfig["proxies"] == nil {
				existingConfig["proxies"] = proxyConfig["proxies"]
			} else {
				// 如果已有代理配置，更新它
				existingProxies := existingConfig["proxies"].(map[string]interface{})
				existingProxies["default"] = proxyConfig["proxies"].(map[string]interface{})["default"]
			}

			// 将合并后的配置转换为 JSON
			newConfig, err := json.MarshalIndent(existingConfig, "", "  ")
			if err != nil {
				return fmt.Errorf("生成 docker 配置失败: %v", err)
			}

			// 确保目录存在
			os.MkdirAll("/etc/docker", 0755)

			// 写入新配置
			if err := os.WriteFile(dockerConfigPath, newConfig, 0644); err != nil {
				// 如果写入失败，尝试恢复备份
				if _, err := os.Stat(backupPath); err == nil {
					if backupData, err := os.ReadFile(backupPath); err == nil {
						os.WriteFile(dockerConfigPath, backupData, 0644)
					}
				}
				return fmt.Errorf("设置 docker 代理失败: %v", err)
			}

			return nil
		},
		GetProxy: func() (string, error) {
			data, err := os.ReadFile("/etc/docker/daemon.json")
			if err != nil {
				return "", fmt.Errorf("读取 docker 配置失败: %v", err)
			}
			return "/etc/docker/daemon.json\n" + string(data), nil
		},
	},
	{
		Name: "apt",
		SetProxy: func(addr string) error {
			config := fmt.Sprintf("Acquire::http::Proxy \"http://%s\";\nAcquire::https::Proxy \"http://%s\";\n", addr, addr)
			if err := os.WriteFile("/etc/apt/apt.conf.d/02proxy.conf", []byte(config), 0644); err != nil {
				return fmt.Errorf("设置 apt 代理失败: %v", err)
			}
			return nil
		},
		GetProxy: func() (string, error) {
			data, err := os.ReadFile("/etc/apt/apt.conf.d/02proxy.conf")
			if err != nil {
				return "", fmt.Errorf("读取 apt 代理配置失败: %v", err)
			}
			return "/etc/apt/apt.conf.d/02proxy.conf\n" + string(data), nil
		},
	},
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "set":
		if len(args) != 2 {
			fmt.Println("错误: set 命令需要一个地址参数")
			fmt.Println("用法: proxy-tool set <address>")
			os.Exit(1)
		}
		handleSetCommand(args[1])
	case "get":
		handleGetCommand()
	case "version":
		fmt.Println("proxy_tool v0.1.0")
	default:
		printUsage()
		os.Exit(1)
	}
}

func handleSetCommand(address string) {
	// 验证地址格式
	if !strings.Contains(address, ":") {
		fmt.Println("错误: 地址格式无效，应为 host:port 格式")
		os.Exit(1)
	}

	// 检查是否以 root 权限运行
	if os.Geteuid() != 0 {
		fmt.Println("错误: 需要 root 权限来设置系统代理")
		os.Exit(1)
	}

	// 为每个服务设置代理
	for _, service := range services {
		fmt.Printf("正在为 %s 设置代理...\n", service.Name)
		if err := service.SetProxy(address); err != nil {
			fmt.Printf("设置 %s 代理失败: %v\n", service.Name, err)
			continue
		}
		fmt.Printf("%s 代理设置成功, 请重启 docker 守护进程\n", service.Name)
	}
}

func handleGetCommand() {
	for _, service := range services {
		proxy, err := service.GetProxy()
		if err != nil {
			fmt.Printf("%s: 获取代理设置失败: %v\n", service.Name, err)
			continue
		}
		fmt.Printf("%s: %s\n", service.Name, proxy)
	}
}

func printUsage() {
	fmt.Println("用法: proxy-tool <command> [arguments]")
	fmt.Println("\n命令:")
	fmt.Println("  set <address>    设置系统代理，地址格式为 host:port")
	fmt.Println("  get             显示当前代理设置")
	fmt.Println("  version         显示版本信息")
	fmt.Println("\n示例:")
	fmt.Println("  sudo proxy-tool set localhost:7890")
	fmt.Println("  proxy-tool get")
}

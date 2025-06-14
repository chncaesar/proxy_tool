package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type ProxyType string

const (
	ProxyTypeSystem ProxyType = "system"
	ProxyTypeUser   ProxyType = "user"
)

type ProxyService struct {
	Name        string
	SetProxy    func(string) error
	GetProxy    func() (string, error)
	NeedRestart bool
	NeedRoot    bool
	Type        ProxyType
}

var systemServices = []ProxyService{
	{
		Name: "docker",
		SetProxy: func(addr string) error {
			dockerConfigPath := "/etc/docker/daemon.json"
			backupPath := dockerConfigPath + ".bak"

			// 准备新的代理配置
			proxyConfig := map[string]interface{}{
				"http-proxy":  "http://" + addr,
				"https-proxy": "http://" + addr,
				"no-proxy":    "localhost,127.0.0.1",
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
			existingConfig["proxies"] = proxyConfig

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
			dockerConfigPath := "/etc/docker/daemon.json"
			data, err := os.ReadFile(dockerConfigPath)
			if err != nil {
				return "", fmt.Errorf("读取 docker 配置失败: %v", err)
			}
			return fmt.Sprintf("配置文件: %s\n%s", dockerConfigPath, string(data)), nil
		},
		NeedRestart: true,
		NeedRoot:    true,
	},
	{
		Name: "apt",
		SetProxy: func(addr string) error {
			config := fmt.Sprintf("Acquire::http::proxy \"http://%s\";\nAcquire::https::proxy \"http://%s\";\n", addr, addr)
			if err := os.WriteFile("/etc/apt/apt.conf.d/02proxy.conf", []byte(config), 0644); err != nil {
				return fmt.Errorf("设置 apt 代理失败: %v", err)
			}
			return nil
		},
		GetProxy: func() (string, error) {
			aptConfigPath := "/etc/apt/apt.conf.d/02proxy.conf"
			data, err := os.ReadFile(aptConfigPath)
			if err != nil {
				return "", fmt.Errorf("读取 apt 代理配置失败: %v", err)
			}
			return fmt.Sprintf("配置文件: %s\n%s", aptConfigPath, string(data)), nil
		},
		NeedRestart: false,
		NeedRoot:    true,
	},
}

var userServices = []ProxyService{
	{
		Name: "npm",
		SetProxy: func(addr string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("获取用户主目录失败: %v", err)
			}
			npmrcPath := homeDir + "/.npmrc"
			backupPath := npmrcPath + ".bak"

			// 准备新的代理配置
			proxyConfig := fmt.Sprintf(`# Proxy Tool 配置的 npm 代理
	proxy=http://%s
	https-proxy=http://%s
	`, addr, addr)

			// 读取现有配置
			var existingContent string
			if _, err := os.Stat(npmrcPath); err == nil {
				// 备份当前配置
				currentConfig, err := os.ReadFile(npmrcPath)
				if err != nil {
					return fmt.Errorf("读取 npm 配置失败: %v", err)
				}
				if err := os.WriteFile(backupPath, currentConfig, 0644); err != nil {
					return fmt.Errorf("备份 npm 配置失败: %v", err)
				}
				fmt.Printf("已备份 npm 配置文件到: %s\n", backupPath)

				existingContent = string(currentConfig)
			}

			// 处理现有配置
			var newContent string
			if existingContent != "" {
				// 使用字符串替换而不是正则表达式
				marker := "# Proxy Tool 配置的 npm 代理"
				if strings.Contains(existingContent, marker) {
					// 找到标记的位置
					start := strings.Index(existingContent, marker)
					// 找到下一个空行的位置
					end := strings.Index(existingContent[start:], "\n\n")
					if end == -1 {
						end = len(existingContent)
					} else {
						end = start + end + 2
					}
					// 替换配置部分
					newContent = existingContent[:start] + proxyConfig + existingContent[end:]
				} else {
					newContent = existingContent + "\n" + proxyConfig
				}
			} else {
				newContent = proxyConfig
			}

			// 写入新配置
			if err := os.WriteFile(npmrcPath, []byte(newContent), 0644); err != nil {
				// 如果写入失败，尝试恢复备份
				if _, err := os.Stat(backupPath); err == nil {
					if backupData, err := os.ReadFile(backupPath); err == nil {
						os.WriteFile(npmrcPath, backupData, 0644)
					}
				}
				return fmt.Errorf("设置 npm 代理失败: %v", err)
			}
			return nil
		},
		GetProxy: func() (string, error) {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("获取用户主目录失败: %v", err)
			}
			npmrcPath := homeDir + "/.npmrc"
			if _, err := os.Stat(npmrcPath); err != nil {
				return "未找到 npm 配置文件", nil
			}
			data, err := os.ReadFile(npmrcPath)
			if err != nil {
				return "", fmt.Errorf("读取 npm 配置失败: %v", err)
			}
			content := string(data)
			marker := "# Proxy Tool 配置的 npm 代理"
			if !strings.Contains(content, marker) {
				return "未找到 npm 代理配置", nil
			}
			// 找到标记的位置
			start := strings.Index(content, marker)
			// 找到下一个空行的位置
			end := strings.Index(content[start:], "\n\n")
			if end == -1 {
				end = len(content)
			} else {
				end = start + end + 2
			}
			return fmt.Sprintf("配置文件: %s\n%s", npmrcPath, content[start:end]), nil
		},
		NeedRestart: false,
		NeedRoot:    false,
		Type:        ProxyTypeUser,
	},
	{
		Name: "env",
		SetProxy: func(addr string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("获取用户主目录失败: %v", err)
			}
			proxyConfig := fmt.Sprintf(`
# Proxy Tool 配置的环境变量
export http_proxy="http://%s"
export https_proxy="http://%s"
export all_proxy="socks5://%s"

`, addr, addr, addr)

			shellConfigs := []string{
				homeDir + "/.bashrc",
				homeDir + "/.zshrc",
				homeDir + "/.bash_profile",
			}
			var configModified bool
			for _, configFile := range shellConfigs {
				if _, err := os.Stat(configFile); err != nil {
					continue
				}
				content, err := os.ReadFile(configFile)
				if err != nil {
					continue
				}
				backupPath := configFile + ".bak"
				_ = os.WriteFile(backupPath, content, 0644)

				contentStr := string(content)
				// 使用字符串替换而不是正则表达式
				marker := "# Proxy Tool 配置的环境变量"
				if strings.Contains(contentStr, marker) {
					// 找到标记的位置
					start := strings.Index(contentStr, marker)
					// 找到下一个空行的位置
					end := strings.Index(contentStr[start:], "\n\n")
					if end == -1 {
						end = len(contentStr)
					} else {
						end = start + end + 2
					}
					// 替换配置部分
					newContent := contentStr[:start] + proxyConfig + contentStr[end:]
					if err := os.WriteFile(configFile, []byte(newContent), 0644); err != nil {
						continue
					}
				} else {
					if err := os.WriteFile(configFile, append(content, []byte(proxyConfig)...), 0644); err != nil {
						continue
					}
				}
				configModified = true
				fmt.Printf("已更新配置文件: %s\n", configFile)
			}
			if !configModified {
				return fmt.Errorf("未找到任何可用的 shell 配置文件")
			}
			fmt.Println("环境变量代理配置已更新，请重新加载配置文件或重启终端以生效")
			fmt.Println("可以使用以下命令重新加载配置：")
			fmt.Println("  source ~/.bashrc  # 如果使用 bash")
			fmt.Println("  source ~/.zshrc   # 如果使用 zsh")
			return nil
		},
		GetProxy: func() (string, error) {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("获取用户主目录失败: %v", err)
			}
			shellConfigs := []string{
				homeDir + "/.bashrc",
				homeDir + "/.zshrc",
				homeDir + "/.bash_profile",
			}
			var result strings.Builder
			for _, configFile := range shellConfigs {
				if _, err := os.Stat(configFile); err != nil {
					continue
				}
				content, err := os.ReadFile(configFile)
				if err != nil {
					continue
				}
				contentStr := string(content)
				marker := "# Proxy Tool 配置的环境变量"
				if strings.Contains(contentStr, marker) {
					// 找到标记的位置
					start := strings.Index(contentStr, marker)
					// 找到下一个空行的位置
					end := strings.Index(contentStr[start:], "\n\n")
					if end == -1 {
						end = len(contentStr)
					} else {
						end = start + end + 2
					}
					// 提取配置部分
					config := contentStr[start:end]
					result.WriteString(fmt.Sprintf("配置文件 %s:\n%s\n", configFile, config))
				}
			}
			if result.Len() == 0 {
				return "未找到环境变量代理配置", nil
			}
			return result.String(), nil
		},
		NeedRestart: false,
		NeedRoot:    false,
		Type:        ProxyTypeUser,
	},
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
				return "", fmt.Errorf("获取 git http 代理设置失败: %v", err)
			}
			cmd2 := exec.Command("git", "config", "--global", "https.proxy")
			output2, err := cmd2.Output()
			if err != nil {
				return "", fmt.Errorf("获取 git https 代理设置失败: %v", err)
			}
			return fmt.Sprintf("Git 代理设置:\nhttp.proxy=%s\nhttps.proxy=%s",
				strings.TrimSpace(string(output)),
				strings.TrimSpace(string(output2))), nil
		},
		NeedRestart: false,
		NeedRoot:    false,
		Type:        ProxyTypeUser,
	},
	{
		Name: "docker",
		SetProxy: func(addr string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("获取用户主目录失败: %v", err)
			}
			dockerConfigPath := homeDir + "/.docker/config.json"
			backupPath := dockerConfigPath + ".bak"

			// 准备新的代理配置
			proxyConfig := map[string]interface{}{
				"default": map[string]interface{}{
					"httpProxy":  "http://" + addr,
					"httpsProxy": "http://" + addr,
					"noProxy":    "localhost,127.0.0.1",
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
			existingConfig["proxies"] = proxyConfig

			// 将合并后的配置转换为 JSON
			newConfig, err := json.MarshalIndent(existingConfig, "", "  ")
			if err != nil {
				return fmt.Errorf("生成 docker 配置失败: %v", err)
			}
			// 确保目录存在
			os.MkdirAll(homeDir+"/.docker", 0755)

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
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("获取用户主目录失败: %v", err)
			}
			dockerConfigPath := homeDir + "/.docker/config.json"
			data, err := os.ReadFile(dockerConfigPath)
			if err != nil {
				return "", fmt.Errorf("读取 docker 配置失败: %v", err)
			}
			return fmt.Sprintf("配置文件: %s\n%s", dockerConfigPath, string(data)), nil
		},
		NeedRestart: false,
		NeedRoot:    false,
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
		if len(args) < 2 || len(args) > 3 {
			fmt.Println("错误: set 命令参数不正确")
			fmt.Println("用法: proxy-tool set [--type=system|user] <address>")
			os.Exit(1)
		}

		var proxyType ProxyType
		var address string

		if len(args) == 3 {
			// 处理 --type 参数
			if !strings.HasPrefix(args[1], "--type=") {
				fmt.Println("错误: 无效的参数格式")
				fmt.Println("用法: proxy-tool set [--type=system|user] <address>")
				os.Exit(1)
			}
			typeStr := strings.TrimPrefix(args[1], "--type=")
			if typeStr != string(ProxyTypeSystem) && typeStr != string(ProxyTypeUser) {
				fmt.Println("错误: type 参数必须是 system 或 user")
				os.Exit(1)
			}
			proxyType = ProxyType(typeStr)
			address = args[2]
		} else {
			// 默认设置为系统代理
			proxyType = ProxyTypeSystem
			address = args[1]
		}

		handleSetCommand(address, proxyType)
	case "get":
		handleGetCommand()
	case "version":
		fmt.Println("proxy_tool v0.1.0")
	default:
		printUsage()
		os.Exit(1)
	}
}

func handleSetCommand(address string, proxyType ProxyType) {
	// 验证地址格式
	if !strings.Contains(address, ":") {
		fmt.Println("错误: 地址格式无效，应为 host:port 格式")
		os.Exit(1)
	}

	// 根据代理类型选择服务
	var services []ProxyService
	if proxyType == ProxyTypeSystem {
		services = systemServices
		// 检查 root 权限
		if os.Geteuid() != 0 {
			fmt.Println("错误: 设置系统代理需要 root 权限")
			os.Exit(1)
		}
	} else {
		services = userServices
	}

	// 设置代理
	for _, service := range services {
		fmt.Printf("正在为 %s 设置代理...\n", service.Name)
		if err := service.SetProxy(address); err != nil {
			fmt.Printf("设置 %s 代理失败: %v\n", service.Name, err)
			continue
		}
		fmt.Printf("%s 代理设置成功\n", service.Name)
		if service.NeedRestart {
			fmt.Printf("注意: %s 需要重启才能生效\n", service.Name)
		}
	}
}

func handleGetCommand() {
	allServices := append(systemServices, userServices...)

	for _, service := range allServices {
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
	fmt.Println("  set [--type=system|user] <address>    设置代理，地址格式为 host:port")
	fmt.Println("                                        --type 参数可选，默认为 system")
	fmt.Println("  get                                   显示当前代理设置")
	fmt.Println("  version                               显示版本信息")
	fmt.Println("\n示例:")
	fmt.Println("  sudo proxy-tool set --type=system localhost:7890  # 设置系统代理")
	fmt.Println("  proxy-tool set --type=user localhost:7890        # 设置用户代理")
	fmt.Println("  proxy-tool get                                   # 查看所有代理设置")
}

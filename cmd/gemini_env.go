package cmd

import (
	"fmt"
	"runtime"
	"strconv"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/spf13/cobra"
)

// getExportFileExample 返回当前系统保存并加载配置文件的命令示例
func getExportFileExample() string {
	switch runtime.GOOS {
	case "windows":
		return "ccs gc > gemini-env.ps1\n     . .\\gemini-env.ps1"
	default:
		return "ccs gc > /tmp/gemini-env.sh\n     source /tmp/gemini-env.sh"
	}
}

var geminiEnvCmd = &cobra.Command{
	Use:   "env [编号或配置名称]",
	Short: "输出 Gemini 配置的环境变量 export 语句",
	Long: `输出指定 Gemini 配置的环境变量设置语句。

如果不指定参数，默认输出当前选中的配置。

使用方式:
  1. 直接执行查看输出:
     ccs gc              # 输出当前配置
     ccs gc 2            # 输出编号为 2 的配置
     ccs gc mygemini     # 输出名为 mygemini 的配置

  2. 加载到当前 shell:
     ` + config.GetEnvCommandExample() + `  # 加载环境变量

  3. 保存到文件:
     ` + getExportFileExample(),
	Aliases: []string{"export", "e"},
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		var provider *config.Provider
		var configName string

		if len(args) == 0 {
			// 无参数：使用当前选中的配置
			provider = manager.GetCurrentProviderForApp("gemini")
			if provider == nil {
				return fmt.Errorf("未找到当前选中的 Gemini 配置，请先使用 'ccs gemini switch <配置名>' 切换或添加配置")
			}
			configName = provider.Name
		} else {
			// 有参数：尝试按编号或名称查找
			arg := args[0]

			// 尝试解析为编号
			if index, err := strconv.Atoi(arg); err == nil {
				providers := manager.ListProvidersForApp("gemini")
				if index < 1 || index > len(providers) {
					return fmt.Errorf("编号 %d 超出范围，当前共有 %d 个配置", index, len(providers))
				}
				provider = &providers[index-1]
				configName = provider.Name
			} else {
				// 按名称查找
				var lookupErr error
				provider, lookupErr = manager.GetProviderForApp("gemini", arg)
				if lookupErr != nil {
					return fmt.Errorf("未找到配置: %s", arg)
				}
				configName = arg
			}
		}

		// 生成 export 语句
		exportScript, err := config.GenerateGeminiEnvExport(provider, configName)
		if err != nil {
			return fmt.Errorf("生成 export 语句失败: %w", err)
		}

		// 输出到 stdout（用户可通过 eval $(ccs gc) 加载）
		fmt.Print(exportScript)

		return nil
	},
}

func init() {
	geminiCmd.AddCommand(geminiEnvCmd)

	// 将 env 命令注册为 gemini 的默认命令（当执行 'ccs gc' 时）
	// 通过设置 RunE 来实现
	geminiCmd.RunE = geminiEnvCmd.RunE
	geminiCmd.Args = geminiEnvCmd.Args
}

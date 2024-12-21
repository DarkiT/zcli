package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/darkit/zcli"
)

// ServiceApp 定义应用服务结构体
type ServiceApp struct {
	svc       *zcli.Service
	logger    *slog.Logger
	isRunning bool
	config    struct {
		Port    string
		Mode    string
		Workers int
	}
}

// NewServiceApp 创建新的服务应用实例
func NewServiceApp() (*ServiceApp, error) {
	app := &ServiceApp{
		logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})),
	}

	// 设置构建信息
	buildInfo := zcli.NewBuildInfo().
		SetDebug(true).
		SetVersion("1.0.0").
		SetBuildTime(time.Now())

	// 创建服务实例
	svc, err := zcli.New(&zcli.Options{
		Name:        "myapp",
		DisplayName: "我的测试服务",
		Description: "这是一个服务示例程序",
		Version:     "1.0.0",
		Language:    "zh",
		Logo: `
 _____ ___ _    ___   
|_   _/ __| |  |_ _|  
  | || (__| |__ | |   
  |_| \___|____|___|  
`,
		Run:       app.run,
		Stop:      app.stop,
		BuildInfo: buildInfo,
	})
	if err != nil {
		return nil, fmt.Errorf("创建服务失败: %w", err)
	}
	app.svc = svc

	// 设置参数
	if err := app.setupParameters(); err != nil {
		return nil, fmt.Errorf("设置参数失败: %w", err)
	}

	// 添加自定义命令
	if err := app.setupCommands(); err != nil {
		return nil, fmt.Errorf("设置命令失败: %w", err)
	}

	return app, nil
}

// setupParameters 配置服务参数
func (app *ServiceApp) setupParameters() error {
	pm := app.svc.ParamManager()

	// 添加配置文件参数
	if err := pm.AddParam(&zcli.Parameter{
		Name:        "config",
		Short:       "c",
		Long:        "config",
		Description: "配置文件路径",
		Required:    false,
		Type:        "string",
	}); err != nil {
		return fmt.Errorf("添加config参数失败: %w", err)
	}

	// 添加端口参数
	if err := pm.AddParam(&zcli.Parameter{
		Name:        "port",
		Short:       "p",
		Long:        "port",
		Description: "服务监听端口",
		Default:     "8080",
		Type:        "string",
		Validate: func(val string) error {
			if val == "0" {
				return fmt.Errorf("端口不能为0")
			}
			return nil
		},
	}); err != nil {
		return fmt.Errorf("添加port参数失败: %w", err)
	}

	// 添加运行模式参数
	if err := pm.AddParam(&zcli.Parameter{
		Name:        "mode",
		Short:       "m",
		Long:        "mode",
		Description: "运行模式",
		Default:     "prod",
		EnumValues:  []string{"dev", "test", "prod"},
		Type:        "string",
	}); err != nil {
		return fmt.Errorf("添加mode参数失败: %w", err)
	}

	// 添加工作线程参数
	if err := pm.AddParam(&zcli.Parameter{
		Name:        "workers",
		Short:       "w",
		Long:        "workers",
		Description: "工作线程数",
		Default:     "5",
		Type:        "string",
		Validate: func(val string) error {
			if val == "0" {
				return fmt.Errorf("工作线程数不能为0")
			}
			return nil
		},
	}); err != nil {
		return fmt.Errorf("添加workers参数失败: %w", err)
	}

	return nil
}

// setupCommands 设置自定义命令
func (app *ServiceApp) setupCommands() error {
	pm := app.svc.ParamManager()

	// 添加版本检查命令
	pm.AddCommand("version", "打印版本信息", func() {
		fmt.Printf("Version: %s\n", app.svc.GetVersion())
	}, true)

	// 添加配置检查命令
	pm.AddCommand("check", "检查配置", func() {
		app.logger.Info("正在检查配置...")
	}, false)

	return nil
}

// loadConfig 加载配置
func (app *ServiceApp) loadConfig() error {
	pm := app.svc.ParamManager()

	// 加载参数到配置结构
	app.config.Port = pm.GetString("port")
	app.config.Mode = pm.GetString("mode")
	app.config.Workers = pm.GetInt("workers")

	return nil
}

// run 服务运行主函数
func (app *ServiceApp) run() error {
	// 加载配置
	if err := app.loadConfig(); err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	app.isRunning = true
	app.logger.Info("服务开始运行...",
		"port", app.config.Port,
		"mode", app.config.Mode,
		"workers", app.config.Workers)

	// 根据模式设置不同的行为
	if app.config.Mode == "dev" {
		app.svc.EnableDebug()
		app.logger.Info("运行在开发模式")
	}

	// 主服务循环
	for app.isRunning {
		if app.svc.IsDebug() {
			app.logger.Debug("服务正在运行...")
		}
		time.Sleep(time.Second * 5)
	}

	return nil
}

// stop 服务停止函数
func (app *ServiceApp) stop() error {
	app.logger.Info("服务准备停止...")
	app.isRunning = false

	// 清理资源
	if err := app.svc.Close(); err != nil {
		return fmt.Errorf("关闭服务失败: %w", err)
	}

	return nil
}

func main() {
	// 创建服务应用
	app, err := NewServiceApp()
	if err != nil {
		slog.Error("初始化失败", "error", err)
		os.Exit(1)
	}

	// 运行服务
	if err := app.svc.Run(); err != nil {
		slog.Error("服务运行失败", "error", err)
		os.Exit(1)
	}
}

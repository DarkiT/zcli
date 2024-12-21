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
}

// NewServiceApp 创建新的服务应用实例
func NewServiceApp() (*ServiceApp, error) {
	app := &ServiceApp{
		logger: slog.Default(),
	}

	// 设置构建信息
	buildInfo := zcli.NewBuildInfo().SetDebug(true).SetVersion("1.0.0").SetBuildTime(time.Now())

	// 创建服务实例
	svc, err := zcli.New(&zcli.Options{
		Name:        "myapp",
		DisplayName: "我的测试服务",
		Description: "这是一个服务示例程序",
		Version:     "1.0.0",
		Language:    "zh",
		Logo: `

 ___                                       
    / / /__  ___/                          
   / /    / /   ___      ___     //  ___   
  / /    / /  //   ) ) //   ) ) // ((   ) )
 / /    / /  //   / / //   / / //   \ \    
/ /___ / /  ((___/ / ((___/ / // //   ) )  
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

// run 服务运行主函数
func (app *ServiceApp) run() error {
	app.isRunning = true
	app.logger.Info("服务开始运行...")

	// 获取参数值
	port := app.svc.ParamManager().GetString("port")
	mode := app.svc.ParamManager().GetString("mode")

	// 使用获取到的参数值
	app.logger.Info(fmt.Sprintf("服务配置 - 端口: %s, 模式: %s", port, mode))

	// 根据模式设置不同的行为
	if mode == "dev" {
		app.logger.Info("运行在开发模式")
	}

	app.svc.SetConfigValue("lastStartTime", time.Now().Unix())

	for app.isRunning {
		time.Sleep(time.Second)
	}
	return nil
}

// stop 服务停止函数
func (app *ServiceApp) stop() error {
	app.logger.Info("服务准备停止...")
	app.isRunning = false
	return nil
}

func main() {
	app, err := NewServiceApp()
	if err != nil {
		slog.Error("初始化失败: %v", err)
		os.Exit(1)
	}

	if err := app.svc.Run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

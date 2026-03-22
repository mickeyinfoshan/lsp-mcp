// Package main MCP-LSP桥接服务主程序
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mickeyinfoshan/lsp-mcp/internal/config"
	"github.com/mickeyinfoshan/lsp-mcp/internal/mcp"
)

var (
	// 版本信息
	version   = "1.0.0"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	// 定义命令行参数
	var (
		configPath  = flag.String("config", "", "配置文件路径 (默认: ./config/config.yaml)")
		showVersion = flag.Bool("version", false, "显示版本信息")
		showHelp    = flag.Bool("help", false, "显示帮助信息")
	)

	flag.Parse()

	// 显示版本信息
	if *showVersion {
		printVersion()
		return
	}

	// 显示帮助信息
	if *showHelp {
		printHelp()
		return
	}

	// 设置默认配置文件路径
	if *configPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatalf("获取工作目录失败: %v", err)
		}
		*configPath = filepath.Join(wd, "config", "config.yaml")
	}

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 设置日志格式
	setupLogging(cfg.Logging)

	log.Printf("MCP-LSP桥接服务启动中... (版本: %s)", version)
	log.Printf("配置文件: %s", *configPath)
	log.Printf("支持的语言: %v", getSupportedLanguages(cfg))

	// 创建MCP服务器
	mcpServer, err := mcp.NewServer(cfg)
	if err != nil {
		log.Fatalf("创建MCP服务器失败: %v", err)
	}

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动服务器的goroutine
	errorChan := make(chan error, 1)
	go func() {
		if err := mcpServer.Serve(); err != nil {
			errorChan <- fmt.Errorf("MCP服务器运行失败: %w", err)
		}
	}()

	log.Println("MCP-LSP桥接服务已启动，等待请求...")

	// 等待信号或错误
	select {
	case sig := <-sigChan:
		log.Printf("收到信号 %v，正在关闭服务...", sig)
	case err := <-errorChan:
		log.Printf("服务器错误: %v", err)
	}

	// 优雅关闭
	log.Println("正在关闭MCP-LSP桥接服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := mcpServer.Shutdown(ctx); err != nil {
		log.Printf("关闭服务器时出错: %v", err)
	}

	log.Println("MCP-LSP桥接服务已关闭")
}

// printVersion 打印版本信息
func printVersion() {
	fmt.Printf("MCP-LSP Bridge Service\n")
	fmt.Printf("Version: %s\n", version)
	fmt.Printf("Build Time: %s\n", buildTime)
	fmt.Printf("Git Commit: %s\n", gitCommit)
}

// printHelp 打印帮助信息
func printHelp() {
	fmt.Printf("MCP-LSP Bridge Service - 基于MCP协议的LSP桥接服务\n\n")
	fmt.Printf("用法:\n")
	fmt.Printf("  %s [选项]\n\n", os.Args[0])
	fmt.Printf("选项:\n")
	fmt.Printf("  -config string\n")
	fmt.Printf("        配置文件路径 (默认: ./config/config.yaml)\n")
	fmt.Printf("  -version\n")
	fmt.Printf("        显示版本信息\n")
	fmt.Printf("  -help\n")
	fmt.Printf("        显示此帮助信息\n\n")
	fmt.Printf("描述:\n")
	fmt.Printf("  MCP-LSP桥接服务提供了一个基于MCP (Model Context Protocol) 的接口，\n")
	fmt.Printf("  用于与各种LSP (Language Server Protocol) 服务器通信。\n")
	fmt.Printf("  支持查找变量定义、获取代码补全等功能。\n\n")
	fmt.Printf("支持的工具:\n")
	fmt.Printf("  - find_definition: 查找变量、函数或类的定义位置\n")
	fmt.Printf("  - get_supported_languages: 获取支持的编程语言列表\n")
	fmt.Printf("  - get_session_info: 获取LSP会话信息和性能指标\n\n")
	fmt.Printf("配置文件示例:\n")
	fmt.Printf("  请参考 config/config.yaml 文件中的配置示例\n\n")
	fmt.Printf("更多信息:\n")
	fmt.Printf("  项目文档: docs/\n")
	fmt.Printf("  GitHub: https://github.com/mark3labs/mcp-go\n")
}

// setupLogging 设置日志格式
func setupLogging(loggingConfig *config.LoggingConfig) {
	// 设置日志标志
	logFlags := log.LstdFlags
	if loggingConfig.Format == "json" {
		// JSON格式日志可能需要不同的处理
		logFlags |= log.Lshortfile
	} else {
		// 文本格式日志
		logFlags |= log.Lshortfile
	}

	log.SetFlags(logFlags)

	// 设置日志前缀
	log.SetPrefix("[MCP-LSP] ")

	// 如果需要输出到文件
	if loggingConfig.FileOutput && loggingConfig.FilePath != "" {
		// 创建日志目录
		logDir := filepath.Dir(loggingConfig.FilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			log.Printf("创建日志目录失败: %v", err)
			return
		}

		// 打开日志文件
		logFile, err := os.OpenFile(loggingConfig.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Printf("打开日志文件失败: %v", err)
			return
		}

		// 设置日志输出到文件
		log.SetOutput(logFile)
		log.Printf("日志输出已设置到文件: %s", loggingConfig.FilePath)
	}
}

// getSupportedLanguages 获取支持的语言列表
func getSupportedLanguages(cfg *config.Config) []string {
	languages := make([]string, 0, len(cfg.LSPServers))
	for languageID := range cfg.LSPServers {
		languages = append(languages, languageID)
	}
	return languages
}

// Package main MCP-LSP bridge service main program
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
	// Version information
	version   = "1.0.0"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	// Define command-line flags
	var (
		configPath  = flag.String("config", "", "Configuration file path (default: ./config/config.yaml)")
		showVersion = flag.Bool("version", false, "Show version information")
		showHelp    = flag.Bool("help", false, "Show help information")
	)

	flag.Parse()

	// Show version information
	if *showVersion {
		printVersion()
		return
	}

	// Show help information
	if *showHelp {
		printHelp()
		return
	}

	// Set default configuration file path
	if *configPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get working directory: %v", err)
		}
		*configPath = filepath.Join(wd, "config", "config.yaml")
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration file: %v", err)
	}

	// Set up logging format
	setupLogging(cfg.Logging)

	log.Printf("MCP-LSP bridge service starting... (version: %s)", version)
	log.Printf("Configuration file: %s", *configPath)
	log.Printf("Supported languages: %v", getSupportedLanguages(cfg))

	// Create MCP server
	mcpServer, err := mcp.NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server goroutine
	errorChan := make(chan error, 1)
	go func() {
		if err := mcpServer.Serve(); err != nil {
			errorChan <- fmt.Errorf("MCP server runtime failed: %w", err)
		}
	}()

	log.Println("MCP-LSP bridge service started, waiting for requests...")

	// Wait for signal or error
	select {
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down service...", sig)
	case err := <-errorChan:
		log.Printf("Server error: %v", err)
	}

	// Graceful shutdown
	log.Println("Shutting down MCP-LSP bridge service...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := mcpServer.Shutdown(ctx); err != nil {
		log.Printf("Error while shutting down server: %v", err)
	}

	log.Println("MCP-LSP bridge service stopped")
}

// printVersion prints version information
func printVersion() {
	fmt.Printf("MCP-LSP Bridge Service\n")
	fmt.Printf("Version: %s\n", version)
	fmt.Printf("Build Time: %s\n", buildTime)
	fmt.Printf("Git Commit: %s\n", gitCommit)
}

// printHelp prints help information
func printHelp() {
	fmt.Printf("MCP-LSP Bridge Service - LSP bridge service based on the MCP protocol\n\n")
	fmt.Printf("Usage:\n")
	fmt.Printf("  %s [options]\n\n", os.Args[0])
	fmt.Printf("Options:\n")
	fmt.Printf("  -config string\n")
	fmt.Printf("        Configuration file path (default: ./config/config.yaml)\n")
	fmt.Printf("  -version\n")
	fmt.Printf("        Show version information\n")
	fmt.Printf("  -help\n")
	fmt.Printf("        Show this help information\n\n")
	fmt.Printf("Description:\n")
	fmt.Printf("  The MCP-LSP bridge service provides an interface based on MCP (Model Context Protocol),\n")
	fmt.Printf("  used to communicate with various LSP (Language Server Protocol) servers.\n")
	fmt.Printf("  It supports features such as finding variable definitions and getting code completions.\n\n")
	fmt.Printf("Supported tools:\n")
	fmt.Printf("  - find_definition: Find the definition location of a variable, function, or class\n")
	fmt.Printf("  - get_supported_languages: Get the list of supported programming languages\n")
	fmt.Printf("  - get_session_info: Get LSP session information and performance metrics\n\n")
	fmt.Printf("Configuration example:\n")
	fmt.Printf("  Please refer to the example configuration in config/config.yaml\n\n")
	fmt.Printf("More information:\n")
	fmt.Printf("  Project documentation: docs/\n")
	fmt.Printf("  GitHub: https://github.com/mark3labs/mcp-go\n")
}

// setupLogging sets up logging format
func setupLogging(loggingConfig *config.LoggingConfig) {
	// Set log flags
	logFlags := log.LstdFlags
	if loggingConfig.Format == "json" {
		// JSON format logging may require different handling
		logFlags |= log.Lshortfile
	} else {
		// Text format logging
		logFlags |= log.Lshortfile
	}

	log.SetFlags(logFlags)

	// Set log prefix
	log.SetPrefix("[MCP-LSP] ")

	// If output to file is needed
	if loggingConfig.FileOutput && loggingConfig.FilePath != "" {
		// Create log directory
		logDir := filepath.Dir(loggingConfig.FilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			log.Printf("Failed to create log directory: %v", err)
			return
		}

		// Open log file
		logFile, err := os.OpenFile(loggingConfig.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Printf("Failed to open log file: %v", err)
			return
		}

		// Set log output to file
		log.SetOutput(logFile)
		log.Printf("Log output set to file: %s", loggingConfig.FilePath)
	}
}

// getSupportedLanguages gets the list of supported languages
func getSupportedLanguages(cfg *config.Config) []string {
	languages := make([]string, 0, len(cfg.LSPServers))
	for languageID := range cfg.LSPServers {
		languages = append(languages, languageID)
	}
	return languages
}

package otelcli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tobert/otel-cli/mcpserver"
	"github.com/tobert/otel-cli/otlpserver"
)

var mcpSvr struct {
	port          int
	projectRoot   string
	maxSpans      int
	retentionTime string
	allowOrigins  string
}

func serverMCPCmd(config *Config) *cobra.Command {
	cmd := cobra.Command{
		Use:   "mcp",
		Short: "Run a Model Context Provider server",
		Long: `Start a Model Context Provider (MCP) server that collects and serves OpenTelemetry trace data 
in a format optimized for consumption by coding agents.

The MCP server provides:
- A WebSocket endpoint for real-time updates at /ws
- REST API endpoints:
  - /api/traces - List all traces
  - /api/trace/{id} - Get a specific trace
  - /api/files - List all files with spans
  - /api/file/{path} - Get traces for a specific file
  - /api/spans/search - Search across traces

Examples:
  # Start an MCP server on port 8080 with default settings
  otel-cli server mcp --port 8080

  # Specify project root for accurate code mapping
  otel-cli server mcp --project-root /home/user/projects/myapp

  # Configure trace retention
  otel-cli server mcp --retention 24h --max-spans 10000`,
		RunE: doMCPServer,
	}

	addCommonParams(&cmd, config)
	cmd.Flags().IntVar(&mcpSvr.port, "port", 8080, "port for the MCP server")
	cmd.Flags().StringVar(&mcpSvr.projectRoot, "project-root", "", "root directory of the project, for code mapping")
	cmd.Flags().IntVar(&mcpSvr.maxSpans, "max-spans", 10000, "maximum number of spans to store")
	cmd.Flags().StringVar(&mcpSvr.retentionTime, "retention", "1h", "retention time for traces (e.g. 1h, 24h, 7d)")
	cmd.Flags().StringVar(&mcpSvr.allowOrigins, "allow-origins", "*", "comma-separated list of allowed origins for CORS")

	return &cmd
}

func doMCPServer(cmd *cobra.Command, args []string) error {
	conf := getConfig(cmd.Context())

	var retention time.Duration
	if mcpSvr.retentionTime != "" {
		var err error
		retention, err = time.ParseDuration(mcpSvr.retentionTime)
		if err != nil {
			return fmt.Errorf("invalid retention time format: %w", err)
		}
	}

	allowedOrigins := []string{"*"} // Default to allow all
	if mcpSvr.allowOrigins != "" {
		allowedOrigins = splitAllowedOrigins(mcpSvr.allowOrigins)
	}

	if mcpSvr.projectRoot == "" {
		pwd, err := os.Getwd()
		if err != nil {
			conf.SoftFail("Failed to get current directory: %v", err)
		} else {
			mcpSvr.projectRoot = pwd
		}
	}

	mcpConfig := &mcpserver.MCPConfig{
		Port:           mcpSvr.port,
		ProjectRoot:    mcpSvr.projectRoot,
		MaxSpans:       mcpSvr.maxSpans,
		RetentionTime:  retention,
		AllowedOrigins: allowedOrigins,
	}

	mcp := mcpserver.NewMCPServer(mcpConfig)

	go runServer(&conf, mcp.HandleSpan, func(otlpserver.OtlpServer) {})
	go mcp.StartMCPServer()

	time.Sleep(time.Millisecond * 10) // avoid race on conf.Endpoint the worst way

	conf.SoftLog("MCP server running on port %d", mcpSvr.port)
	conf.SoftLog("OTLP Server running on %s", conf.Endpoint)
	conf.SoftLog("Project root: %s", mcpSvr.projectRoot)

	waitForInterrupt(conf) // keep server running until signal
	return nil
}

// Split comma-separated list of allowed origins
func splitAllowedOrigins(origins string) []string {
	if origins == "*" {
		return []string{"*"}
	}
	return strings.Split(origins, ",")
}

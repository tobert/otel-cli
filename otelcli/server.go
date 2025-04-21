package otelcli

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tobert/otel-cli/otlpserver"
)

const defaultOtlpEndpoint = "grpc://localhost:4317"
const spanBgSockfilename = "otel-cli-background.sock"

func serverCmd(config *Config) *cobra.Command {
	cmd := cobra.Command{
		Use:   "server",
		Short: "run an embedded OTLP server",
		Long:  "Run otel-cli as an OTLP server. See subcommands.",
	}

	cmd.AddCommand(serverJsonCmd(config))
	cmd.AddCommand(serverTuiCmd(config))
	cmd.AddCommand(serverMCPCmd(config))

	return &cmd
}

// runServer runs the server on either grpc or http and blocks until the server
// stops or is killed.
func runServer(config *Config, cb otlpserver.Callback, stop otlpserver.Stopper) {
	// unlike the rest of otel-cli, server should default to localhost:4317
	if config.Endpoint == "" {
		config.Endpoint = defaultOtlpEndpoint
		Diag.EndpointSource = "default"
	}
	endpointURL, _ := config.ParseEndpoint()

	var cs otlpserver.OtlpServer
	if config.Protocol != "grpc" &&
		(strings.HasPrefix(config.Protocol, "http/") ||
			endpointURL.Scheme == "http") {
		cs = otlpserver.NewServer("http", cb, stop)
	} else if config.Protocol == "https" || endpointURL.Scheme == "https" {
		config.SoftFail("https server is not supported yet, please raise an issue")
	} else {
		cs = otlpserver.NewServer("grpc", cb, stop)
	}

	defer cs.Stop()
	cs.ListenAndServe(endpointURL.Host)
}

// Helper to wait for interrupt signal
func waitForInterrupt(config Config) {
	config.SoftLog("Server running. Press Ctrl+C to stop...")
	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, os.Interrupt, syscall.SIGTERM)
	<-interruptChannel
	config.SoftLog("\nshutting down MCP server...")
}

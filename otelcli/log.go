package otelcli

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/tobert/otel-cli/otlpclient"
)

// logCmd represents the log command
func logCmd(config *Config) *cobra.Command {
	cmd := cobra.Command{
		Use:   "log",
		Short: "create an OpenTelemetry log record and send it",
		Long: `Create an OpenTelemetry log record as specified and send it along.

Example:
	otel-cli log \
		--service "my-application" \
		--body "User login successful" \
		--severity INFO \
		--attrs "user.id=123,action=login"
`,
		Run: doLog,
	}

	cmd.Flags().SortFlags = false

	addCommonParams(&cmd, config)
	addServiceParams(&cmd, config)
	addLogParams(&cmd, config)
	addAttrParams(&cmd, config)
	addClientParams(&cmd, config)

	return &cmd
}

func doLog(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	config := getConfig(ctx)
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(config.GetTimeout()))
	defer cancel()

	// Create logs-specific client
	client := otlpclient.NewGrpcLogsClient(config)
	ctx, err := client.Start(ctx)
	config.SoftFailIfErr(err)

	// Create log record
	logRecord := config.NewProtobufLogRecord()

	// Send log
	ctx, err = client.UploadLogs(ctx, logRecord)
	config.SoftFailIfErr(err)

	// Cleanup
	_, err = client.Stop(ctx)
	config.SoftFailIfErr(err)
}

// addLogParams adds log-specific parameters to the command.
func addLogParams(cmd *cobra.Command, config *Config) {
	defaults := DefaultConfig()

	cmd.Flags().StringVar(&config.LogBody, "body", defaults.LogBody, "log message body")
	cmd.Flags().StringVar(&config.LogSeverity, "severity", defaults.LogSeverity, "log severity level (TRACE, DEBUG, INFO, WARN, ERROR, FATAL)")
}

package otelcli

import (
	"github.com/tobert/otel-cli/otlpclient"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
)

// NewProtobufLogRecord creates a new log record and populates it with information
// from the config struct.
func (c Config) NewProtobufLogRecord() *logspb.LogRecord {
	logRecord := otlpclient.NewProtobufLogRecord()

	// Set the log body
	if c.LogBody != "" {
		logRecord.Body = &commonpb.AnyValue{
			Value: &commonpb.AnyValue_StringValue{StringValue: c.LogBody},
		}
	}

	// Set severity
	logRecord.SeverityNumber = otlpclient.LogSeverityStringToInt(c.LogSeverity)
	logRecord.SeverityText = c.LogSeverity

	// Add attributes from config
	logRecord.Attributes = otlpclient.StringMapAttrsToProtobuf(c.Attributes)

	return logRecord
}

package otlpclient

// Implements just enough sugar on the OTel Protocol Buffers log definition
// to support otel-cli and no more.

import (
	"time"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
)

// NewProtobufLogRecord returns an initialized OpenTelemetry protobuf LogRecord.
func NewProtobufLogRecord() *logspb.LogRecord {
	now := time.Now()
	logRecord := logspb.LogRecord{
		TimeUnixNano:           uint64(now.UnixNano()),
		ObservedTimeUnixNano:   uint64(now.UnixNano()),
		SeverityNumber:         logspb.SeverityNumber_SEVERITY_NUMBER_INFO,
		SeverityText:           "INFO",
		Body:                   &commonpb.AnyValue{},
		Attributes:             []*commonpb.KeyValue{},
		DroppedAttributesCount: 0,
		Flags:                  0,
		TraceId:                []byte{},
		SpanId:                 []byte{},
	}

	return &logRecord
}

// LogSeverityStringToInt takes a supported string log severity and returns the otel
// constant for it. Returns default of INFO on no match.
func LogSeverityStringToInt(severity string) logspb.SeverityNumber {
	switch severity {
	case "TRACE", "trace":
		return logspb.SeverityNumber_SEVERITY_NUMBER_TRACE
	case "DEBUG", "debug":
		return logspb.SeverityNumber_SEVERITY_NUMBER_DEBUG
	case "INFO", "info":
		return logspb.SeverityNumber_SEVERITY_NUMBER_INFO
	case "WARN", "warn":
		return logspb.SeverityNumber_SEVERITY_NUMBER_WARN
	case "ERROR", "error":
		return logspb.SeverityNumber_SEVERITY_NUMBER_ERROR
	case "FATAL", "fatal":
		return logspb.SeverityNumber_SEVERITY_NUMBER_FATAL
	default:
		return logspb.SeverityNumber_SEVERITY_NUMBER_INFO
	}
}

// LogSeverityIntToString takes an integer/constant protobuf log severity value
// and returns the string representation used in otel-cli.
func LogSeverityIntToString(severity logspb.SeverityNumber) string {
	switch severity {
	case logspb.SeverityNumber_SEVERITY_NUMBER_TRACE:
		return "TRACE"
	case logspb.SeverityNumber_SEVERITY_NUMBER_DEBUG:
		return "DEBUG"
	case logspb.SeverityNumber_SEVERITY_NUMBER_INFO:
		return "INFO"
	case logspb.SeverityNumber_SEVERITY_NUMBER_WARN:
		return "WARN"
	case logspb.SeverityNumber_SEVERITY_NUMBER_ERROR:
		return "ERROR"
	case logspb.SeverityNumber_SEVERITY_NUMBER_FATAL:
		return "FATAL"
	default:
		return "INFO"
	}
}

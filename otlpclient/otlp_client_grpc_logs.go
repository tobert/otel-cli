package otlpclient

import (
	"context"
	"fmt"
	"time"

	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// GrpcLogsClient holds the state for gRPC connections for logs.
type GrpcLogsClient struct {
	conn   *grpc.ClientConn
	client collogspb.LogsServiceClient
	config OTLPConfig
}

// NewGrpcLogsClient returns a fresh GrpcLogsClient ready to Start.
func NewGrpcLogsClient(config OTLPConfig) *GrpcLogsClient {
	c := GrpcLogsClient{config: config}
	return &c
}

// Start configures and starts the connection to the gRPC server in the background.
func (gc *GrpcLogsClient) Start(ctx context.Context) (context.Context, error) {
	var err error
	endpointURL := gc.config.GetLogsEndpoint()
	host := endpointURL.Hostname()
	if endpointURL.Port() != "" {
		host = host + ":" + endpointURL.Port()
	}

	grpcOpts := []grpc.DialOption{}

	if gc.config.GetInsecure() {
		grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(credentials.NewTLS(gc.config.GetTlsConfig())))
	}

	gc.conn, err = grpc.DialContext(ctx, host, grpcOpts...)
	if err != nil {
		return ctx, fmt.Errorf("could not connect to gRPC/OTLP logs endpoint: %w", err)
	}

	gc.client = collogspb.NewLogsServiceClient(gc.conn)

	return ctx, nil
}

// UploadLogs takes a list of protobuf log records and sends them out.
func (gc *GrpcLogsClient) UploadLogs(ctx context.Context, logRecord *logspb.LogRecord) (context.Context, error) {
	// add headers onto the request
	headers := gc.config.GetHeaders()
	if len(headers) > 0 {
		md := metadata.New(headers)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	resourceAttrs, err := resourceAttributes(ctx, gc.config.GetServiceName())
	if err != nil {
		return ctx, err
	}

	rls := []*logspb.ResourceLogs{
		{
			Resource: &resourcepb.Resource{
				Attributes: resourceAttrs,
			},
			ScopeLogs: []*logspb.ScopeLogs{
				{
					Scope: &commonpb.InstrumentationScope{
						Name:                   "github.com/tobert/otel-cli",
						Version:                gc.config.GetVersion(),
						Attributes:             []*commonpb.KeyValue{},
						DroppedAttributesCount: 0,
					},
					LogRecords: []*logspb.LogRecord{logRecord},
					SchemaUrl:  semconv.SchemaURL,
				},
			},
			SchemaUrl: semconv.SchemaURL,
		},
	}

	req := collogspb.ExportLogsServiceRequest{ResourceLogs: rls}

	_, err = gc.client.Export(ctx, &req)
	if err != nil {
		return SaveError(ctx, time.Now(), err)
	}

	return ctx, nil
}

// Stop closes the connection to the gRPC server.
func (gc *GrpcLogsClient) Stop(ctx context.Context) (context.Context, error) {
	return ctx, gc.conn.Close()
}

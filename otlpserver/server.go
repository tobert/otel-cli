// otlpserver is an OTLP server with HTTP and gRPC backends available.
// It takes a lot of shortcuts to keep things simple and is not intended
// to be used as a serious OTLP service. Primarily it is for the test
// suite and also supports the otel-cli server features.
package otlpserver

import (
	"context"
	"crypto/tls"
	"net"

	colv1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Callback is a type for the function passed to newServer that is
// called for each incoming span.
type Callback func(context.Context, *tracepb.Span, []*tracepb.Span_Event, *tracepb.ResourceSpans, map[string]string, map[string]string) bool

// Stopper is the function passed to newServer to be called when the
// server is shut down.
type Stopper func(OtlpServer)

// OtlpServer abstracts the minimum interface required for an OTLP
// server to be either HTTP or gRPC (but not both, for now).
type OtlpServer interface {
	ListenAndServe(otlpEndpoint string)
	Serve(listener net.Listener) error
	Stop()
	StopWait()
}

// NewServer will start the requested server protocol, one of grpc, http/protobuf,
// and http/json. Optional TLS configuration can be provided for gRPC servers.
func NewServer(protocol string, cb Callback, stop Stopper, tlsConf ...*tls.Config) OtlpServer {
	switch protocol {
	case "grpc":
		// if TLS config is provided, convert to gRPC credentials
		var opts []grpc.ServerOption
		if len(tlsConf) > 0 && tlsConf[0] != nil {
			creds := credentials.NewTLS(tlsConf[0])
			opts = append(opts, grpc.Creds(creds))
		}
		return NewGrpcServer(cb, stop, opts...)
	case "http":
		return NewHttpServer(cb, stop)
	}

	return nil
}

// doCallback unwraps the OTLP service request and calls the callback
// for each span in the request.
func doCallback(ctx context.Context, cb Callback, req *colv1.ExportTraceServiceRequest, headers map[string]string, serverMeta map[string]string) bool {
	rss := req.GetResourceSpans()
	for _, resource := range rss {
		scopeSpans := resource.GetScopeSpans()
		for _, ss := range scopeSpans {
			for _, span := range ss.GetSpans() {
				events := span.GetEvents()
				if events == nil {
					events = []*tracepb.Span_Event{}
				}

				done := cb(ctx, span, events, resource, headers, serverMeta)
				if done {
					return true
				}
			}
		}
	}

	return false
}

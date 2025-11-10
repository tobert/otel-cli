package otlpserver

import (
	"bytes"
	"context"
	"encoding/csv"
	"log"
	"net"
	"sync"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// grpcServerState holds the shared state for all signal services.
type grpcServerState struct {
	server        *grpc.Server
	traceCallback TraceCallback
	logCallback   LogCallback
	stoponce      sync.Once
	stopper       chan struct{}
	stopdone      chan struct{}
	doneonce      sync.Once
}

// GrpcServer is a gRPC/OTLP server handle.
type GrpcServer struct {
	state *grpcServerState
	coltracepb.UnimplementedTraceServiceServer
}

// grpcLogsService handles logs exports.
type grpcLogsService struct {
	state *grpcServerState
	collogspb.UnimplementedLogsServiceServer
}

// NewGrpcServer takes a callback and stop function and returns a Server ready
// to run with .Serve(). Optional grpc.ServerOption arguments can be provided
// for TLS configuration and other server options.
func NewGrpcServer(cb TraceCallback, stop Stopper, opts ...grpc.ServerOption) *GrpcServer {
	state := &grpcServerState{
		server:        grpc.NewServer(opts...),
		traceCallback: cb,
		stopper:       make(chan struct{}),
		stopdone:      make(chan struct{}, 1),
	}

	s := &GrpcServer{state: state}

	coltracepb.RegisterTraceServiceServer(state.server, s)
	collogspb.RegisterLogsServiceServer(state.server, &grpcLogsService{state: state})

	// single place to stop the server, used by timeout and max-spans
	go func() {
		<-state.stopper
		stop(s)
		state.server.GracefulStop()
	}()

	return s
}

// SetLogCallback sets the log callback for the server.
func (gs *GrpcServer) SetLogCallback(cb LogCallback) {
	gs.state.logCallback = cb
}

// ServeGRPC takes a listener and starts the GRPC server on that listener.
// Blocks until Stop() is called.
func (gs *GrpcServer) Serve(listener net.Listener) error {
	err := gs.state.server.Serve(listener)
	gs.state.stopdone <- struct{}{}
	return err
}

// ListenAndServeGRPC starts a TCP listener then starts the GRPC server using
// ServeGRPC for you.
func (gs *GrpcServer) ListenAndServe(otlpEndpoint string) {
	listener, err := net.Listen("tcp", otlpEndpoint)
	if err != nil {
		log.Fatalf("failed to listen on OTLP endpoint %q: %s", otlpEndpoint, err)
	}
	if err := gs.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}

// Stop sends a value to the server shutdown goroutine so it stops GRPC
// and calls the stop function given to newServer. Safe to call multiple times.
func (gs *GrpcServer) Stop() {
	gs.state.stoponce.Do(func() {
		gs.state.stopper <- struct{}{}
	})
}

// StopWait stops the server and waits for it to affirm shutdown.
func (gs *GrpcServer) StopWait() {
	gs.Stop()
	gs.state.doneonce.Do(func() {
		<-gs.state.stopdone
	})
}

// Export implements the gRPC server interface for exporting trace messages.
func (gs *GrpcServer) Export(ctx context.Context, req *coltracepb.ExportTraceServiceRequest) (*coltracepb.ExportTraceServiceResponse, error) {
	// OTLP/gRPC headers are passed in metadata, copy them to serverMeta
	// for now. This isn't ideal but gets them exposed to the test suite.
	headers := make(map[string]string)
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		for mdk := range md {
			vals := md.Get(mdk)
			buf := bytes.NewBuffer([]byte{})
			csv.NewWriter(buf).WriteAll([][]string{vals})
			headers[mdk] = buf.String()
		}
	}

	done := doCallback(ctx, gs.state.traceCallback, req, headers, map[string]string{"proto": "grpc"})
	if done {
		go gs.StopWait()
	}
	return &coltracepb.ExportTraceServiceResponse{}, nil
}

// Export implements the gRPC server interface for exporting log records.
func (ls *grpcLogsService) Export(ctx context.Context, req *collogspb.ExportLogsServiceRequest) (*collogspb.ExportLogsServiceResponse, error) {
	// only process if we have a log callback set
	if ls.state.logCallback == nil {
		return &collogspb.ExportLogsServiceResponse{}, nil
	}

	// OTLP/gRPC headers are passed in metadata, copy them to serverMeta
	headers := make(map[string]string)
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		for mdk := range md {
			vals := md.Get(mdk)
			buf := bytes.NewBuffer([]byte{})
			csv.NewWriter(buf).WriteAll([][]string{vals})
			headers[mdk] = buf.String()
		}
	}

	done := doLogCallback(ctx, ls.state.logCallback, req, headers, map[string]string{"proto": "grpc"})
	if done {
		// need to call StopWait on the GrpcServer, not directly on state
		// so we create a temporary GrpcServer wrapper
		gs := &GrpcServer{state: ls.state}
		go gs.StopWait()
	}
	return &collogspb.ExportLogsServiceResponse{}, nil
}

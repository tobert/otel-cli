package otlpserver

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/proto"
)

// HttpServer is a handle for otlp over http/protobuf.
type HttpServer struct {
	server        *http.Server
	traceCallback TraceCallback
	logCallback   LogCallback
}

// NewServer takes a callback and stop function and returns a Server ready
// to run with .Serve().
func NewHttpServer(cb TraceCallback, stop Stopper) *HttpServer {
	s := HttpServer{
		server:        &http.Server{},
		traceCallback: cb,
	}

	s.server.Handler = &s

	return &s
}

// ServeHTTP routes requests to the appropriate handler based on URL path.
func (hs *HttpServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Route based on OTLP specification paths
	switch req.RequestURI {
	case "/v1/traces":
		hs.handleTraces(rw, req)
	case "/v1/logs":
		hs.handleLogs(rw, req)
	default:
		// For backwards compatibility, treat unspecified paths as traces
		hs.handleTraces(rw, req)
	}
}

// handleTraces processes trace export requests.
func (hs *HttpServer) handleTraces(rw http.ResponseWriter, req *http.Request) {
	data, err := io.ReadAll(req.Body)
	if err != nil {
		log.Fatalf("Error while reading request body: %s", err)
	}

	msg := coltracepb.ExportTraceServiceRequest{}
	switch req.Header.Get("Content-Type") {
	case "application/x-protobuf":
		proto.Unmarshal(data, &msg)
	case "application/json":
		json.Unmarshal(data, &msg)
	default:
		rw.WriteHeader(http.StatusNotAcceptable)
		return
	}

	meta := map[string]string{
		"method":       req.Method,
		"proto":        req.Proto,
		"content-type": req.Header.Get("Content-Type"),
		"host":         req.Host,
		"uri":          req.RequestURI,
	}

	headers := make(map[string]string)
	for k := range req.Header {
		headers[k] = req.Header.Get(k)
	}

	done := doCallback(req.Context(), hs.traceCallback, &msg, headers, meta)
	if done {
		go hs.StopWait()
	}
}

// handleLogs processes log export requests.
func (hs *HttpServer) handleLogs(rw http.ResponseWriter, req *http.Request) {
	if hs.logCallback == nil {
		rw.WriteHeader(http.StatusOK)
		return
	}

	data, err := io.ReadAll(req.Body)
	if err != nil {
		log.Fatalf("Error while reading request body: %s", err)
	}

	msg := collogspb.ExportLogsServiceRequest{}
	switch req.Header.Get("Content-Type") {
	case "application/x-protobuf":
		proto.Unmarshal(data, &msg)
	case "application/json":
		json.Unmarshal(data, &msg)
	default:
		rw.WriteHeader(http.StatusNotAcceptable)
		return
	}

	meta := map[string]string{
		"method":       req.Method,
		"proto":        req.Proto,
		"content-type": req.Header.Get("Content-Type"),
		"host":         req.Host,
		"uri":          req.RequestURI,
	}

	headers := make(map[string]string)
	for k := range req.Header {
		headers[k] = req.Header.Get(k)
	}

	done := doLogCallback(req.Context(), hs.logCallback, &msg, headers, meta)
	if done {
		go hs.StopWait()
	}
}

// ServeHttp takes a listener and starts the HTTP server on that listener.
// Blocks until Stop() is called.
func (hs *HttpServer) Serve(listener net.Listener) error {
	err := hs.server.Serve(listener)
	return err
}

// ListenAndServeHttp starts a TCP listener then starts the HTTP server using
// ServeHttp for you.
func (hs *HttpServer) ListenAndServe(otlpEndpoint string) {
	listener, err := net.Listen("tcp", otlpEndpoint)
	if err != nil {
		log.Fatalf("failed to listen on OTLP endpoint %q: %s", otlpEndpoint, err)
	}
	if err := hs.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}

// Stop closes the http server and all active connections immediately.
func (hs *HttpServer) Stop() {
	hs.server.Close()
}

// StopWait stops the http server gracefully.
func (hs *HttpServer) StopWait() {
	hs.server.Shutdown(context.Background())
}

// SetLogCallback sets the log callback for the server.
func (hs *HttpServer) SetLogCallback(cb LogCallback) {
	hs.logCallback = cb
}

// mcpserver is a Model Context Provider server that collects and serves
// OpenTelemetry trace data in a format optimized for consumption by coding
// agents like Claude.
package mcpserver

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

// MCPConfig holds the configuration for the MCP server
type MCPConfig struct {
	Port           int
	ProjectRoot    string
	MaxSpans       int
	RetentionTime  time.Duration
	AllowedOrigins []string
}

// MCPServer is the main server struct for the MCP server
type MCPServer struct {
	store       *TraceStore
	analyzer    *CodeAnalyzer
	httpServer  *http.Server
	upgrader    websocket.Upgrader
	clients     map[*websocket.Conn]bool
	clientsLock sync.Mutex
	config      *MCPConfig
}

// TraceStore manages traces with added context for AI consumption
type TraceStore struct {
	traces      map[string]*TraceData         // traceID -> trace data
	spansByFile map[string][]*CodeSpanContext // file path -> spans touching this file
	lock        sync.RWMutex
	maxSpans    int
	retention   time.Duration
}

// TraceData holds complete trace information
type TraceData struct {
	TraceID      string
	RootSpan     *SpanData
	Spans        map[string]*SpanData // spanID -> span data
	Files        map[string]bool      // files touched by this trace
	StartTime    time.Time
	EndTime      time.Time
	Status       string
	ErrorMessage string
}

// SpanData enriches raw spans with context
type SpanData struct {
	SpanProto    *tracepb.Span
	Events       []*tracepb.Span_Event
	ParentID     string
	Children     []string
	FileContexts []*CodeSpanContext
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
}

// CodeSpanContext links spans to source code
type CodeSpanContext struct {
	FilePath     string
	LineStart    int
	LineEnd      int
	FunctionName string
	SpanID       string
	TraceID      string
	Operation    string // "read", "write", "exec", etc.
	CodeSnapshot string // The actual code relevant to this span
}

// NewMCPServer creates a new MCP server with the given configuration
func NewMCPServer(config *MCPConfig) *MCPServer {
	store := &TraceStore{
		traces:      make(map[string]*TraceData),
		spansByFile: make(map[string][]*CodeSpanContext),
		maxSpans:    config.MaxSpans,
		retention:   config.RetentionTime,
	}

	analyzer := &CodeAnalyzer{
		projectRoot: config.ProjectRoot,
	}

	return &MCPServer{
		store:    store,
		analyzer: analyzer,
		clients:  make(map[*websocket.Conn]bool),
		config:   config,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				for _, origin := range config.AllowedOrigins {
					if origin == "*" {
						return true
					}
					if origin == r.Header.Get("Origin") {
						return true
					}
				}
				return false
			},
		},
	}
}

// HandleSpan processes incoming spans from the OTLP server
func (mcp *MCPServer) HandleSpan(ctx context.Context, span *tracepb.Span, events []*tracepb.Span_Event,
	rs *tracepb.ResourceSpans, headers map[string]string, meta map[string]string) bool {

	spanData := &SpanData{
		SpanProto: span,
		Events:    events,
		ParentID:  hex.EncodeToString(span.ParentSpanId),
		StartTime: time.Unix(0, int64(span.StartTimeUnixNano)),
		EndTime:   time.Unix(0, int64(span.EndTimeUnixNano)),
	}
	spanData.Duration = spanData.EndTime.Sub(spanData.StartTime)
	spanData.FileContexts = mcp.analyzer.AnalyzeSpan(span, events)

	mcp.store.AddSpan(spanData)

	mcp.notifyClients(spanData)

	return false // don't stop server
}

// StartMCPServer starts the MCP HTTP server. Blocks forever.
func (mcp *MCPServer) StartMCPServer() {
	mux := http.NewServeMux()

	mux.HandleFunc("/ws", mcp.handleWebsocket)

	mux.HandleFunc("/api/traces", mcp.handleListTraces)
	mux.HandleFunc("/api/trace/", mcp.handleGetTrace)
	mux.HandleFunc("/api/files", mcp.handleListFiles)
	mux.HandleFunc("/api/file/", mcp.handleGetFileTraces)
	mux.HandleFunc("/api/spans/search", mcp.handleSearchSpans)

	mux.Handle("/", GetUIHandler())

	mcp.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", mcp.config.Port),
		Handler: mux,
	}

	if err := mcp.httpServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("error starting MCP server: %v", err)
	}
}

// notifyClients sends a message to all connected WebSocket clients
func (mcp *MCPServer) notifyClients(spanData *SpanData) {
	message := WebSocketMessage{
		Type:    "new_span",
		SpanID:  hex.EncodeToString(spanData.SpanProto.SpanId),
		TraceID: hex.EncodeToString(spanData.SpanProto.TraceId),
	}

	messageJSON, _ := json.Marshal(message)

	mcp.clientsLock.Lock()
	defer mcp.clientsLock.Unlock()

	for client := range mcp.clients {
		if err := client.WriteMessage(websocket.TextMessage, messageJSON); err != nil {
			client.Close()
			delete(mcp.clients, client)
		}
	}
}

package mcpserver

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type    string `json:"type"`
	Message string `json:"message,omitempty"`
	SpanID  string `json:"span_id,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
}

// TraceResponse represents the detailed response for a trace
type TraceResponse struct {
	TraceID      string                 `json:"traceId"`
	StartTime    interface{}            `json:"startTime"`
	EndTime      interface{}            `json:"endTime"`
	Status       string                 `json:"status"`
	ErrorMessage string                 `json:"message,omitempty"`
	Files        map[string]bool        `json:"files"`
	Spans        map[string]SpanResponse `json:"spans"`
}

// SpanResponse represents a simplified span for API responses
type SpanResponse struct {
	Name        string              `json:"name"`
	ParentID    string              `json:"parentId,omitempty"`
	Children    []string            `json:"children,omitempty"`
	StartTime   interface{}         `json:"startTime"`
	Duration    int64               `json:"durationMs"`
	FileContexts []FileContextResponse `json:"fileContexts,omitempty"`
}

// FileContextResponse represents file context information for API responses
type FileContextResponse struct {
	FilePath     string `json:"filePath"`
	FunctionName string `json:"functionName,omitempty"`
	Operation    string `json:"operation"`
	LineStart    int    `json:"lineStart"`
	LineEnd      int    `json:"lineEnd"`
	CodeSnippet  string `json:"codeSnippet,omitempty"`
}

// SearchRequest defines parameters for trace queries
type SearchRequest struct {
	Query      string   `json:"query"`      // Natural language query
	Files      []string `json:"files"`      // Files of interest
	TimeRange  string   `json:"timeRange"`  // Time range like "1h", "24h"
	ErrorsOnly bool     `json:"errorsOnly"` // Only return traces with errors
	Limit      int      `json:"limit"`      // Max results
}

// SearchResponse provides AI-friendly trace data
type SearchResponse struct {
	Traces      []*TraceDigest          `json:"traces"`
	FileInsights map[string]*FileInsight `json:"fileInsights"`
	Summary     string                  `json:"summary"`
}
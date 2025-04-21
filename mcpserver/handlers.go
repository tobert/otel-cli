package mcpserver

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// handleWebsocket handles WebSocket connections
func (mcp *MCPServer) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	conn, err := mcp.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to websocket: %v", err)
		return
	}
	
	// Register the new client
	mcp.clientsLock.Lock()
	mcp.clients[conn] = true
	mcp.clientsLock.Unlock()
	
	// Handle client disconnection
	go func() {
		defer conn.Close()
		
		// Wait for client to disconnect or send a message
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				mcp.clientsLock.Lock()
				delete(mcp.clients, conn)
				mcp.clientsLock.Unlock()
				break
			}
		}
	}()
	
	// Send initial event to confirm connection
	message := WebSocketMessage{
		Type:    "connected",
		Message: "Connected to MCP server",
	}
	
	conn.WriteJSON(message)
}

// handleListTraces lists all traces in the store
func (mcp *MCPServer) handleListTraces(w http.ResponseWriter, r *http.Request) {
	traces := mcp.store.ListTraces()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(traces)
}

// handleGetTrace returns a specific trace by ID
func (mcp *MCPServer) handleGetTrace(w http.ResponseWriter, r *http.Request) {
	traceID := strings.TrimPrefix(r.URL.Path, "/api/trace/")
	
	trace := mcp.store.GetTrace(traceID)
	if trace == nil {
		http.Error(w, "Trace not found", http.StatusNotFound)
		return
	}
	
	// Create a response object with selected information
	response := TraceResponse{
		TraceID:      trace.TraceID,
		StartTime:    trace.StartTime,
		EndTime:      trace.EndTime,
		Status:       trace.Status,
		ErrorMessage: trace.ErrorMessage,
		Files:        trace.Files,
		Spans:        make(map[string]SpanResponse),
	}
	
	// Add simplified span data
	for id, span := range trace.Spans {
		spanResp := SpanResponse{
			Name:      span.SpanProto.Name,
			ParentID:  span.ParentID,
			Children:  span.Children,
			StartTime: span.StartTime,
			Duration:  span.Duration.Milliseconds(),
		}
		
		// Add file contexts for this span
		for _, ctx := range span.FileContexts {
			ctxResp := FileContextResponse{
				FilePath:     ctx.FilePath,
				FunctionName: ctx.FunctionName,
				Operation:    ctx.Operation,
				LineStart:    ctx.LineStart,
				LineEnd:      ctx.LineEnd,
				CodeSnippet:  ctx.CodeSnapshot,
			}
			spanResp.FileContexts = append(spanResp.FileContexts, ctxResp)
		}
		
		response.Spans[id] = spanResp
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleListFiles returns all files that have associated spans
func (mcp *MCPServer) handleListFiles(w http.ResponseWriter, r *http.Request) {
	files := mcp.store.ListFiles()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

// handleGetFileTraces returns all traces associated with a specific file
func (mcp *MCPServer) handleGetFileTraces(w http.ResponseWriter, r *http.Request) {
	filePath := strings.TrimPrefix(r.URL.Path, "/api/file/")
	
	fileTraces := mcp.store.GetFileTraces(filePath)
	if len(fileTraces) == 0 {
		http.Error(w, "File not found in any traces", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fileTraces)
}

// handleSearchSpans handles the search API endpoint
func (mcp *MCPServer) handleSearchSpans(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Process search request
	results := mcp.store.SearchTraces(req)
	
	// Generate a summary if requested
	if req.Query != "" {
		summary := generateSearchSummary(req, results)
		results.Summary = summary
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// generateSearchSummary creates a textual summary of the search results
func generateSearchSummary(req SearchRequest, results *SearchResponse) string {
	if len(results.Traces) == 0 {
		return "No traces found matching your query."
	}
	
	// Basic summary
	summary := ""
	
	if req.ErrorsOnly {
		summary += "Found " + pluralize(len(results.Traces), "trace", "traces") + " with errors"
	} else {
		summary += "Found " + pluralize(len(results.Traces), "trace", "traces")
	}
	
	// Add file context if available
	if len(req.Files) > 0 {
		if len(req.Files) == 1 {
			summary += " involving the file " + req.Files[0]
		} else {
			summary += " involving " + pluralize(len(req.Files), "file", "files")
		}
	}
	
	// Add time range if available
	if req.TimeRange != "" {
		summary += " in the last " + req.TimeRange
	}
	
	summary += "."
	
	// Add error statistics
	var totalErrors int
	for _, trace := range results.Traces {
		totalErrors += trace.ErrorCount
	}
	
	if totalErrors > 0 {
		summary += " Total of " + pluralize(totalErrors, "error", "errors") + " detected."
	}
	
	// Add file insights
	if len(results.FileInsights) > 0 {
		summary += " The analysis covers " + pluralize(len(results.FileInsights), "file", "files") + "."
	}
	
	return summary
}

// pluralize is a helper to correctly pluralize words
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return "1 " + singular
	}
	return strconv.Itoa(count) + " " + plural
}
package mcpserver

import (
	"encoding/hex"
	"log"
	"time"
)

// AddSpan adds a span to the trace store, organizing by trace ID and updating related data
func (store *TraceStore) AddSpan(spanData *SpanData) {
	store.lock.Lock()
	defer store.lock.Unlock()

	traceID := hex.EncodeToString(spanData.SpanProto.TraceId)
	spanID := hex.EncodeToString(spanData.SpanProto.SpanId)

	// Get or create trace
	trace, exists := store.traces[traceID]
	if !exists {
		trace = &TraceData{
			TraceID:  traceID,
			Spans:    make(map[string]*SpanData),
			Files:    make(map[string]bool),
			StartTime: spanData.StartTime,
			EndTime:   spanData.EndTime,
		}
		store.traces[traceID] = trace
	}

	// Update trace start/end times if needed
	if spanData.StartTime.Before(trace.StartTime) {
		trace.StartTime = spanData.StartTime
	}
	if spanData.EndTime.After(trace.EndTime) {
		trace.EndTime = spanData.EndTime
	}

	// Add span to trace
	trace.Spans[spanID] = spanData

	// If span is a root span (no parent), set it as the trace's root span
	if len(spanData.ParentID) == 0 || spanData.ParentID == "0000000000000000" {
		trace.RootSpan = spanData
	} else {
		// Add this span as a child of its parent
		parentID := spanData.ParentID
		if parent, ok := trace.Spans[parentID]; ok {
			parent.Children = append(parent.Children, spanID)
		}
	}

	// Process file contexts
	for _, fileCtx := range spanData.FileContexts {
		filePath := fileCtx.FilePath

		// Add to file index
		store.spansByFile[filePath] = append(store.spansByFile[filePath], fileCtx)
		
		// Mark file as touched by this trace
		trace.Files[filePath] = true

		// Update trace status if this is an error
		if fileCtx.Operation == "error" || fileCtx.Operation == "exception" {
			trace.Status = "error"
		}
	}

	// Clean up old traces if we exceed the maximum
	store.cleanupOldTraces()
}

// cleanupOldTraces removes old traces based on retention time and max spans limit
func (store *TraceStore) cleanupOldTraces() {
	// Skip if no limits are set
	if store.maxSpans <= 0 && store.retention <= 0 {
		return
	}
	
	// Count spans and find old traces
	var oldTraceIDs []string
	var totalSpans int
	now := time.Now()
	
	for id, trace := range store.traces {
		totalSpans += len(trace.Spans)
		
		// Check retention time
		if store.retention > 0 {
			age := now.Sub(trace.EndTime)
			if age > store.retention {
				oldTraceIDs = append(oldTraceIDs, id)
				continue
			}
		}
	}
	
	// If we exceed max spans, remove old traces
	if store.maxSpans > 0 && totalSpans > store.maxSpans {
		// Clean by age if retention wasn't enough
		if len(oldTraceIDs) == 0 {
			// Find the oldest traces
			type traceAge struct {
				id  string
				age time.Time
			}
			
			var ages []traceAge
			for id, trace := range store.traces {
				ages = append(ages, traceAge{id: id, age: trace.EndTime})
			}
			
			// Sort by age (oldest first)
			for i := 0; i < len(ages); i++ {
				for j := i + 1; j < len(ages); j++ {
					if ages[i].age.After(ages[j].age) {
						ages[i], ages[j] = ages[j], ages[i]
					}
				}
			}
			
			// Take enough old traces to get under the limit
			var removed int
			for _, ta := range ages {
				if totalSpans <= store.maxSpans {
					break
				}
				trace := store.traces[ta.id]
				removed += len(trace.Spans)
				totalSpans -= len(trace.Spans)
				oldTraceIDs = append(oldTraceIDs, ta.id)
			}
		}
	}
	
	// Remove the old traces and clean up the file index
	for _, id := range oldTraceIDs {
		trace := store.traces[id]
		
		// Remove from file index
		for file := range trace.Files {
			var newSpans []*CodeSpanContext
			for _, sc := range store.spansByFile[file] {
				if sc.TraceID != id {
					newSpans = append(newSpans, sc)
				}
			}
			
			if len(newSpans) > 0 {
				store.spansByFile[file] = newSpans
			} else {
				delete(store.spansByFile, file)
			}
		}
		
		// Remove the trace
		delete(store.traces, id)
	}
	
	if len(oldTraceIDs) > 0 {
		log.Printf("Removed %d old traces from store", len(oldTraceIDs))
	}
}

// GetTrace returns a specific trace by ID
func (store *TraceStore) GetTrace(traceID string) *TraceData {
	store.lock.RLock()
	defer store.lock.RUnlock()
	
	return store.traces[traceID]
}

// GetSpan returns a specific span by trace ID and span ID
func (store *TraceStore) GetSpan(traceID, spanID string) *SpanData {
	store.lock.RLock()
	defer store.lock.RUnlock()
	
	trace, ok := store.traces[traceID]
	if !ok {
		return nil
	}
	
	return trace.Spans[spanID]
}

// GetFileTraces returns all traces associated with a specific file
func (store *TraceStore) GetFileTraces(filePath string) map[string][]*CodeSpanContext {
	store.lock.RLock()
	defer store.lock.RUnlock()
	
	result := make(map[string][]*CodeSpanContext)
	
	for _, sc := range store.spansByFile[filePath] {
		traceID := sc.TraceID
		result[traceID] = append(result[traceID], sc)
	}
	
	return result
}

// ListFiles returns all files that have associated spans
func (store *TraceStore) ListFiles() []string {
	store.lock.RLock()
	defer store.lock.RUnlock()
	
	var files []string
	for file := range store.spansByFile {
		files = append(files, file)
	}
	
	return files
}

// ListTraces returns summaries of all traces
func (store *TraceStore) ListTraces() []*TraceDigest {
	store.lock.RLock()
	defer store.lock.RUnlock()
	
	var digests []*TraceDigest
	
	for id, trace := range store.traces {
		digest := &TraceDigest{
			TraceID:    id,
			SpanCount:  len(trace.Spans),
			StartTime:  trace.StartTime,
			Duration:   float64(trace.EndTime.Sub(trace.StartTime).Milliseconds()),
		}
		
		// Get name from root span if available
		if trace.RootSpan != nil && trace.RootSpan.SpanProto != nil {
			digest.Name = trace.RootSpan.SpanProto.Name
		}
		
		// Get files
		for file := range trace.Files {
			digest.Files = append(digest.Files, file)
		}
		
		// Count errors
		for _, span := range trace.Spans {
			for _, ctx := range span.FileContexts {
				if ctx.Operation == "error" || ctx.Operation == "exception" {
					digest.ErrorCount++
				}
			}
		}
		
		digests = append(digests, digest)
	}
	
	return digests
}

// SearchTraces performs a search across traces based on the given criteria
func (store *TraceStore) SearchTraces(req SearchRequest) *SearchResponse {
	store.lock.RLock()
	defer store.lock.RUnlock()
	
	var traces []*TraceDigest
	fileInsights := make(map[string]*FileInsight)
	
	// Process file filters
	var fileSet map[string]bool
	if len(req.Files) > 0 {
		fileSet = make(map[string]bool)
		for _, f := range req.Files {
			fileSet[f] = true
		}
	}
	
	// Process time range filter
	var minTime time.Time
	if req.TimeRange != "" {
		duration, err := time.ParseDuration(req.TimeRange)
		if err == nil {
			minTime = time.Now().Add(-duration)
		}
	}
	
	// Collect matching traces
	for id, trace := range store.traces {
		// Skip if outside time range
		if !minTime.IsZero() && trace.EndTime.Before(minTime) {
			continue
		}
		
		// Skip if errors only and no errors
		if req.ErrorsOnly && trace.Status != "error" {
			continue
		}
		
		// Check file filter
		if fileSet != nil {
			hasMatchingFile := false
			for file := range trace.Files {
				if fileSet[file] {
					hasMatchingFile = true
					break
				}
			}
			if !hasMatchingFile {
				continue
			}
		}
		
		// Add to results
		digest := &TraceDigest{
			TraceID:    id,
			SpanCount:  len(trace.Spans),
			StartTime:  trace.StartTime,
			Duration:   float64(trace.EndTime.Sub(trace.StartTime).Milliseconds()),
		}
		
		// Get name from root span if available
		if trace.RootSpan != nil && trace.RootSpan.SpanProto != nil {
			digest.Name = trace.RootSpan.SpanProto.Name
		}
		
		// Get files and build insights
		for file := range trace.Files {
			digest.Files = append(digest.Files, file)
			
			// Build file insights
			insight, exists := fileInsights[file]
			if !exists {
				insight = &FileInsight{
					FilePath: file,
				}
				fileInsights[file] = insight
			}
			
			// Collect hotspots and errors
			for _, span := range trace.Spans {
				for _, ctx := range span.FileContexts {
					if ctx.FilePath == file {
						// Count as error if appropriate
						if ctx.Operation == "error" || ctx.Operation == "exception" {
							digest.ErrorCount++
							insight.ErrorLines = append(insight.ErrorLines, ctx.LineStart)
						}
						
						// Track line hotspots
						insight.HotspotLines = append(insight.HotspotLines, ctx.LineStart)
						
						// Track related files (exclude this file)
						for otherFile := range trace.Files {
							if otherFile != file {
								insight.Related = append(insight.Related, otherFile)
							}
						}
					}
				}
			}
		}
		
		traces = append(traces, digest)
		
		// Limit results if requested
		if req.Limit > 0 && len(traces) >= req.Limit {
			break
		}
	}
	
	// Build response
	response := &SearchResponse{
		Traces:      traces,
		FileInsights: fileInsights,
	}
	
	return response
}

// TraceDigest provides key information about a trace
type TraceDigest struct {
	TraceID      string    `json:"traceId"`
	Name         string    `json:"name"`
	Duration     float64   `json:"durationMs"`
	SpanCount    int       `json:"spanCount"`
	ErrorCount   int       `json:"errorCount"`
	Files        []string  `json:"files"`
	StartTime    time.Time `json:"startTime"`
	KeyEvents    []string  `json:"keyEvents"`
}

// FileInsight provides code-centric insights
type FileInsight struct {
	FilePath     string   `json:"filePath"`
	HotspotLines []int    `json:"hotspotLines"` // Lines with most activity
	ErrorLines   []int    `json:"errorLines"`   // Lines associated with errors
	Related      []string `json:"related"`      // Related files
}

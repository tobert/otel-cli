package mcpserver

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

// CodeAnalyzer extracts code context from spans
type CodeAnalyzer struct {
	projectRoot string
}

// AnalyzeSpan examines a span and its events to extract code context
func (ca *CodeAnalyzer) AnalyzeSpan(span *tracepb.Span, events []*tracepb.Span_Event) []*CodeSpanContext {
	contexts := []*CodeSpanContext{}

	// Extract file paths from span attributes
	fileContexts := ca.extractFilePathsFromSpan(span)
	contexts = append(contexts, fileContexts...)

	// Process each event for file contexts
	for _, event := range events {
		// Check for stack traces in event attributes
		if event.Name == "exception" || strings.Contains(strings.ToLower(event.Name), "error") {
			stackTraceContexts := ca.processStackTrace(event)
			contexts = append(contexts, stackTraceContexts...)
		}
		
		// Look for file operations in event attributes
		fileOpContexts := ca.extractFilePathsFromEvent(event)
		contexts = append(contexts, fileOpContexts...)
	}

	return contexts
}

// extractFilePathsFromSpan looks for file paths in span attributes
func (ca *CodeAnalyzer) extractFilePathsFromSpan(span *tracepb.Span) []*CodeSpanContext {
	contexts := []*CodeSpanContext{}
	
	// Look for file paths in span attributes
	for _, attr := range span.Attributes {
		// Check for common file-related attribute keys
		if strings.Contains(strings.ToLower(attr.Key), "file") || 
		   strings.Contains(strings.ToLower(attr.Key), "path") {
			
			if val := attr.GetValue().GetStringValue(); val != "" {
				if filepath.IsAbs(val) && ca.isFileInProject(val) {
					context := &CodeSpanContext{
						FilePath:  val,
						SpanID:    fmt.Sprintf("%x", span.SpanId),
						TraceID:   fmt.Sprintf("%x", span.TraceId),
						Operation: inferOperationFromSpan(span),
					}
					
					// Read the file to get code context
					ca.enrichWithFileContents(context)
					contexts = append(contexts, context)
				}
			}
		}
	}
	
	return contexts
}

// extractFilePathsFromEvent looks for file paths in event attributes
func (ca *CodeAnalyzer) extractFilePathsFromEvent(event *tracepb.Span_Event) []*CodeSpanContext {
	contexts := []*CodeSpanContext{}
	
	// Look through event attributes for file paths
	for _, attr := range event.Attributes {
		if strings.Contains(strings.ToLower(attr.Key), "file") || 
		   strings.Contains(strings.ToLower(attr.Key), "path") {
			
			if val := attr.GetValue().GetStringValue(); val != "" {
				if filepath.IsAbs(val) && ca.isFileInProject(val) {
					context := &CodeSpanContext{
						FilePath:  val,
						Operation: inferOperationFromEvent(event),
					}
					
					// Read the file to get code context
					ca.enrichWithFileContents(context)
					contexts = append(contexts, context)
				}
			}
		}
	}
	
	return contexts
}

// isFileInProject checks if a file is within the project root
func (ca *CodeAnalyzer) isFileInProject(path string) bool {
	if ca.projectRoot == "" {
		return true
	}
	return strings.HasPrefix(path, ca.projectRoot)
}

// processStackTrace extracts file paths and line numbers from stack traces
func (ca *CodeAnalyzer) processStackTrace(event *tracepb.Span_Event) []*CodeSpanContext {
	contexts := []*CodeSpanContext{}
	
	// Look for stack trace in event attributes
	var stackTrace string
	for _, attr := range event.Attributes {
		if strings.ToLower(attr.Key) == "stack_trace" || strings.ToLower(attr.Key) == "stacktrace" {
			stackTrace = attr.GetValue().GetStringValue()
			break
		}
	}
	
	if stackTrace == "" {
		return contexts
	}
	
	// Parse stack trace using regular expressions
	// Different languages have different stack trace formats, but many contain file:line
	fileLineRegex := regexp.MustCompile(`([\w\/\.\-]+\.[\w]+):(\d+)`)
	matches := fileLineRegex.FindAllStringSubmatch(stackTrace, -1)
	
	for _, match := range matches {
		if len(match) >= 3 {
			filePath := match[1]
			lineNum, _ := strconv.Atoi(match[2])
			
			// Normalize the path and check if it's in the project
			if !filepath.IsAbs(filePath) {
				filePath = filepath.Join(ca.projectRoot, filePath)
			}
			
			if ca.isFileInProject(filePath) {
				context := &CodeSpanContext{
					FilePath:  filePath,
					LineStart: lineNum,
					LineEnd:   lineNum + 10, // Include a few lines of context
					Operation: "exception",
				}
				
				// Read the file to get code context
				ca.enrichWithFileContents(context)
				contexts = append(contexts, context)
			}
		}
	}
	
	return contexts
}

// enrichWithFileContents reads the source file to add code context
func (ca *CodeAnalyzer) enrichWithFileContents(context *CodeSpanContext) {
	if context.FilePath == "" || !fileExists(context.FilePath) {
		return
	}
	
	file, err := os.Open(context.FilePath)
	if err != nil {
		return
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	lineNum := 0
	var codeLines []string
	
	if context.LineStart <= 0 {
		context.LineStart = 1 // Default to start of file
	}
	
	// If we don't know the end line, set a reasonable default
	if context.LineEnd <= context.LineStart {
		context.LineEnd = context.LineStart + 20 // Show about 20 lines
	}
	
	// Cap max lines to avoid huge code snippets
	if context.LineEnd - context.LineStart > 50 {
		context.LineEnd = context.LineStart + 50
	}
	
	for scanner.Scan() {
		lineNum++
		
		// Capture a few lines before the start for context
		if lineNum >= context.LineStart-5 && lineNum <= context.LineEnd {
			codeLines = append(codeLines, fmt.Sprintf("%d: %s", lineNum, scanner.Text()))
		}
		
		if lineNum > context.LineEnd {
			break
		}
	}
	
	context.CodeSnapshot = strings.Join(codeLines, "\n")
	
	// Try to infer function name from code
	ca.inferFunctionName(context)
}

// inferFunctionName attempts to extract the function name from the code
func (ca *CodeAnalyzer) inferFunctionName(context *CodeSpanContext) {
	if context.CodeSnapshot == "" {
		return
	}
	
	// Look for common function/method declarations based on file extension
	ext := strings.ToLower(filepath.Ext(context.FilePath))
	
	// Different regex for different languages
	var funcRegex *regexp.Regexp
	
	switch ext {
	case ".go":
		funcRegex = regexp.MustCompile(`func\s+([A-Za-z0-9_]+)`)
	case ".js", ".ts", ".jsx", ".tsx":
		funcRegex = regexp.MustCompile(`function\s+([A-Za-z0-9_]+)|([A-Za-z0-9_]+)\s*=\s*\(.*\)\s*=>|([A-Za-z0-9_]+)\s*\(.*\)\s*{`)
	case ".py":
		funcRegex = regexp.MustCompile(`def\s+([A-Za-z0-9_]+)`)
	case ".java", ".kt", ".c", ".cpp", ".cc":
		funcRegex = regexp.MustCompile(`[A-Za-z0-9_<>]+\s+([A-Za-z0-9_]+)\s*\(`)
	case ".rb":
		funcRegex = regexp.MustCompile(`def\s+([A-Za-z0-9_]+)`)
	default:
		// Generic function-like pattern
		funcRegex = regexp.MustCompile(`function\s+([A-Za-z0-9_]+)|\s+([A-Za-z0-9_]+)\s*\(`)
	}
	
	matches := funcRegex.FindStringSubmatch(context.CodeSnapshot)
	if len(matches) > 1 {
		for i := 1; i < len(matches); i++ {
			if matches[i] != "" {
				context.FunctionName = matches[i]
				return
			}
		}
	}
}

// inferOperationFromSpan guesses the operation type based on span attributes and name
func inferOperationFromSpan(span *tracepb.Span) string {
	spanName := strings.ToLower(span.Name)
	
	if strings.Contains(spanName, "read") || strings.Contains(spanName, "get") {
		return "read"
	}
	if strings.Contains(spanName, "write") || strings.Contains(spanName, "put") || 
	   strings.Contains(spanName, "save") || strings.Contains(spanName, "update") {
		return "write"
	}
	if strings.Contains(spanName, "exec") || strings.Contains(spanName, "run") {
		return "exec"
	}
	if strings.Contains(spanName, "delete") || strings.Contains(spanName, "remove") {
		return "delete"
	}
	
	// Look at span attributes for more clues
	for _, attr := range span.Attributes {
		attrKey := strings.ToLower(attr.Key)
		if strings.Contains(attrKey, "operation") || strings.Contains(attrKey, "action") {
			if val := attr.GetValue().GetStringValue(); val != "" {
				return strings.ToLower(val)
			}
		}
	}
	
	return "unknown"
}

// inferOperationFromEvent guesses the operation type based on event attributes and name
func inferOperationFromEvent(event *tracepb.Span_Event) string {
	eventName := strings.ToLower(event.Name)
	
	if strings.Contains(eventName, "read") || strings.Contains(eventName, "get") {
		return "read"
	}
	if strings.Contains(eventName, "write") || strings.Contains(eventName, "put") || 
	   strings.Contains(eventName, "save") || strings.Contains(eventName, "update") {
		return "write"
	}
	if strings.Contains(eventName, "exec") || strings.Contains(eventName, "run") {
		return "exec"
	}
	if strings.Contains(eventName, "delete") || strings.Contains(eventName, "remove") {
		return "delete"
	}
	if strings.Contains(eventName, "error") || strings.Contains(eventName, "exception") {
		return "error"
	}
	
	// Look at event attributes for more clues
	for _, attr := range event.Attributes {
		attrKey := strings.ToLower(attr.Key)
		if strings.Contains(attrKey, "operation") || strings.Contains(attrKey, "action") {
			if val := attr.GetValue().GetStringValue(); val != "" {
				return strings.ToLower(val)
			}
		}
	}
	
	return "unknown"
}

// fileExists checks if a file exists and is not a directory
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
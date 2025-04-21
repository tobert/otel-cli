package mcpserver

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed ui
var uiContent embed.FS

// GetUIHandler returns an http.Handler that serves the embedded UI files
func GetUIHandler() http.Handler {
	fsys, err := fs.Sub(uiContent, "ui")
	if err != nil {
		// This should never happen as we're always embedding the ui directory
		panic("failed to create sub-filesystem for UI content: " + err.Error())
	}
	return http.FileServer(http.FS(fsys))
}
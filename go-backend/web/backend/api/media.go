package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

func (h *Handler) registerMediaRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/media/upload", h.handleMediaUpload)
}

// handleMediaUpload accepts multipart file uploads.
// For images it returns inline base64 data URLs; for text files it returns
// the file contents prefixed with the filename; for other binaries it
// returns a descriptive note.
func (h *Handler) handleMediaUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	var refs []string
	for _, files := range r.MultipartForm.File {
		for _, fh := range files {
			f, err := fh.Open()
			if err != nil {
				continue
			}
			data, err := io.ReadAll(io.LimitReader(f, 10<<20)) // 10 MB limit
			f.Close()
			if err != nil {
				continue
			}

			contentType := fh.Header.Get("Content-Type")
			if contentType == "" {
				contentType = http.DetectContentType(data)
			}

			// For images, return as data: URL for inline display / vision models
			if strings.HasPrefix(contentType, "image/") {
				dataURL := "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data)
				refs = append(refs, dataURL)
				continue
			}

			// For text-based files, return content inline
			ext := strings.ToLower(filepath.Ext(fh.Filename))
			textExts := map[string]bool{
				".txt": true, ".md": true, ".json": true, ".csv": true,
				".py": true, ".js": true, ".ts": true, ".go": true,
			}
			if textExts[ext] {
				refs = append(refs, "file:"+fh.Filename+"\n"+string(data))
				continue
			}

			// Binary files: include descriptive note
			refs = append(refs, "file:"+fh.Filename+" (binary, "+fmt.Sprintf("%d", len(data))+" bytes)")
		}
	}

	if len(refs) == 0 {
		http.Error(w, "No files processed", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"refs": refs})
}

// Package server cung cấp HTTP API và giao diện cho LinkHub.
package server

import (
	"fmt"
	"net/http"
	"time"

	"linkhub/internal/store"
	"linkhub/internal/webui"
)

// Server gắn kết kho dữ liệu và HTTP client dùng chung.
type Server struct {
	store  *store.Store
	client *http.Client
}

func New(st *store.Store) *Server {
	return &Server{
		store: st,
		client: &http.Client{
			Timeout: 6 * time.Second,
		},
	}
}

// Handler trả về toàn bộ route API, tài nguyên tĩnh và trang chính.
func (s *Server) Handler() (http.Handler, error) {
	indexHTML, err := webui.Index()
	if err != nil {
		return nil, fmt.Errorf("nạp index.html: %w", err)
	}
	staticFS, err := webui.Static()
	if err != nil {
		return nil, fmt.Errorf("nạp tài nguyên tĩnh: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/links", s.handleList)
	mux.HandleFunc("POST /api/links", s.handleCreate)
	mux.HandleFunc("PUT /api/links/{id}", s.handleUpdate)
	mux.HandleFunc("DELETE /api/links/{id}", s.handleDelete)
	mux.HandleFunc("POST /api/links/{id}/visit", s.handleVisit)
	mux.HandleFunc("POST /api/reorder", s.handleReorder)
	mux.HandleFunc("GET /api/export", s.handleExport)
	mux.HandleFunc("POST /api/import", s.handleImport)
	mux.HandleFunc("GET /api/metadata", s.handleMetadata)
	mux.HandleFunc("POST /api/check", s.handleCheck)

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFS)))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(indexHTML)
	})

	return recoverer(logger(mux)), nil
}

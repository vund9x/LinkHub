package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"linkhub/internal/link"
)

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.List())
}

type linkInput struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Icon        string `json:"icon"`
	Pinned      bool   `json:"pinned"`
}

func (in *linkInput) sanitize() error {
	normalized, err := normalizeURL(in.URL)
	if err != nil {
		return err
	}
	in.URL = normalized
	in.Title = truncate(strings.TrimSpace(in.Title), 120)
	in.Description = truncate(strings.TrimSpace(in.Description), 240)
	in.Category = truncate(strings.TrimSpace(in.Category), 40)
	in.Icon = truncate(strings.TrimSpace(in.Icon), 8)
	if in.Title == "" {
		in.Title = hostLabel(in.URL)
	}
	return nil
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	var in linkInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := in.sanitize(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	created, err := s.store.Create(link.Link{
		Title:       in.Title,
		URL:         in.URL,
		Description: in.Description,
		Category:    in.Category,
		Icon:        in.Icon,
		Pinned:      in.Pinned,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Không thể lưu liên kết")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	var in linkInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := in.sanitize(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	updated, found, err := s.store.Update(r.PathValue("id"), func(l *link.Link) {
		l.Title = in.Title
		l.URL = in.URL
		l.Description = in.Description
		l.Category = in.Category
		l.Icon = in.Icon
		l.Pinned = in.Pinned
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Không thể lưu liên kết")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "Không tìm thấy liên kết")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	found, err := s.store.Delete(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Không thể xóa liên kết")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "Không tìm thấy liên kết")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleVisit(w http.ResponseWriter, r *http.Request) {
	visited, found, err := s.store.RecordVisit(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Không thể ghi nhận lượt truy cập")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "Không tìm thấy liên kết")
		return
	}
	writeJSON(w, http.StatusOK, visited)
}

func (s *Server) handleReorder(w http.ResponseWriter, r *http.Request) {
	var in struct {
		IDs []string `json:"ids"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.store.Reorder(in.IDs); err != nil {
		writeError(w, http.StatusInternalServerError, "Không thể sắp xếp lại")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	payload := map[string]any{
		"version":    1,
		"exportedAt": time.Now().UTC(),
		"links":      s.store.List(),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Không thể xuất dữ liệu")
		return
	}
	filename := "linkhub-" + time.Now().Format("20060102-1504") + ".json"
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Write(data)
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Không thể đọc tệp")
		return
	}

	var file struct {
		Links []link.Link `json:"links"`
	}
	if err := json.Unmarshal(body, &file); err != nil {
		var arr []link.Link
		if err2 := json.Unmarshal(body, &arr); err2 != nil {
			writeError(w, http.StatusBadRequest, "Tệp JSON không hợp lệ")
			return
		}
		file.Links = arr
	}

	cleaned := make([]link.Link, 0, len(file.Links))
	skipped := 0
	for _, l := range file.Links {
		normalized, err := normalizeURL(l.URL)
		if err != nil {
			skipped++
			continue
		}
		l.URL = normalized
		l.Title = truncate(strings.TrimSpace(l.Title), 120)
		l.Description = truncate(strings.TrimSpace(l.Description), 240)
		l.Category = truncate(strings.TrimSpace(l.Category), 40)
		l.Icon = truncate(strings.TrimSpace(l.Icon), 8)
		if l.Title == "" {
			l.Title = hostLabel(l.URL)
		}
		cleaned = append(cleaned, l)
	}

	replace := r.URL.Query().Get("mode") == "replace"
	added, err := s.store.Import(cleaned, replace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Không thể nhập dữ liệu")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"imported": added, "skipped": skipped})
}

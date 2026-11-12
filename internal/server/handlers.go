package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"linkhub/internal/link"
	"linkhub/internal/store"
)

const (
	maxCategoriesPerLink = 12
	maxCategoryName      = 40
)

type linkInput struct {
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	Description string   `json:"description"`
	CategoryIDs []string `json:"categoryIds"`
	Category    string   `json:"category"` // tên danh mục, tương thích API cũ
	Icon        string   `json:"icon"`
	Pinned      bool     `json:"pinned"`
}

func (in *linkInput) sanitize() error {
	normalized, err := normalizeURL(in.URL)
	if err != nil {
		return err
	}
	in.URL = normalized
	in.Title = truncate(strings.TrimSpace(in.Title), 120)
	in.Description = truncate(strings.TrimSpace(in.Description), 240)
	in.Icon = truncate(strings.TrimSpace(in.Icon), 8)
	in.CategoryIDs = cleanCategoryIDs(in.CategoryIDs)
	if in.Title == "" {
		in.Title = hostLabel(in.URL)
	}
	return nil
}

func cleanCategoryIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	seen := make(map[string]bool, len(ids))
	for _, raw := range ids {
		id := truncate(strings.TrimSpace(raw), 32)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
		if len(out) >= maxCategoriesPerLink {
			break
		}
	}
	return out
}

// resolveCategory dùng cho trường category kiểu cũ: tạo danh mục nếu chưa có.
func (s *Server) resolveCategory(in *linkInput) (string, error) {
	name := truncate(strings.TrimSpace(in.Category), maxCategoryName)
	if name == "" {
		return "", nil
	}
	cat, _, err := s.store.CreateCategory(name)
	if err != nil {
		return "", err
	}
	return cat.ID, nil
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.List())
}

func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Categories())
}

type categoryInput struct {
	Name string `json:"name"`
}

func (s *Server) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var in categoryInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	name := truncate(strings.TrimSpace(in.Name), maxCategoryName)
	if name == "" {
		writeError(w, http.StatusBadRequest, "Tên danh mục không được để trống")
		return
	}

	cat, created, err := s.store.CreateCategory(name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Không thể lưu danh mục")
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, cat)
}

func (s *Server) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	var in categoryInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	name := truncate(strings.TrimSpace(in.Name), maxCategoryName)
	if name == "" {
		writeError(w, http.StatusBadRequest, "Tên danh mục không được để trống")
		return
	}

	cat, found, err := s.store.UpdateCategory(r.PathValue("id"), name)
	if errors.Is(err, store.ErrCategoryNameTaken) {
		writeError(w, http.StatusConflict, "Danh mục “"+name+"” đã tồn tại")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Không thể lưu danh mục")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "Không tìm thấy danh mục")
		return
	}
	writeJSON(w, http.StatusOK, cat)
}

func (s *Server) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	found, err := s.store.DeleteCategory(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Không thể xóa danh mục")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "Không tìm thấy danh mục")
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	legacyID, err := s.resolveCategory(&in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Không thể lưu danh mục")
		return
	}
	in.CategoryIDs = appendCategoryID(in.CategoryIDs, legacyID)

	created, err := s.store.Create(link.Link{
		Title:       in.Title,
		URL:         in.URL,
		Description: in.Description,
		CategoryIDs: in.CategoryIDs,
		Icon:        in.Icon,
		Pinned:      in.Pinned,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Không thể lưu liên kết")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func appendCategoryID(ids []string, id string) []string {
	if id == "" || len(ids) >= maxCategoriesPerLink {
		return ids
	}
	for _, existing := range ids {
		if existing == id {
			return ids
		}
	}
	return append(ids, id)
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
	legacyID, err := s.resolveCategory(&in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Không thể lưu danh mục")
		return
	}
	in.CategoryIDs = appendCategoryID(in.CategoryIDs, legacyID)

	updated, found, err := s.store.Update(r.PathValue("id"), func(l *link.Link) {
		l.Title = in.Title
		l.URL = in.URL
		l.Description = in.Description
		l.CategoryIDs = in.CategoryIDs
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
		"version":    2,
		"exportedAt": time.Now().UTC(),
		"categories": s.store.Categories(),
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
		Categories []link.Category    `json:"categories"`
		Links      []store.ImportLink `json:"links"`
	}
	if err := json.Unmarshal(body, &file); err != nil {
		var arr []store.ImportLink
		if err2 := json.Unmarshal(body, &arr); err2 != nil {
			writeError(w, http.StatusBadRequest, "Tệp JSON không hợp lệ")
			return
		}
		file.Links = arr
	}

	cleaned := make([]store.ImportLink, 0, len(file.Links))
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
		l.Category = truncate(strings.TrimSpace(l.Category), maxCategoryName)
		l.CategoryIDs = cleanCategoryIDs(l.CategoryIDs)
		l.Icon = truncate(strings.TrimSpace(l.Icon), 8)
		if l.Title == "" {
			l.Title = hostLabel(l.URL)
		}
		cleaned = append(cleaned, l)
	}

	categories := make([]link.Category, 0, len(file.Categories))
	for _, c := range file.Categories {
		name := truncate(strings.TrimSpace(c.Name), maxCategoryName)
		if name == "" {
			continue
		}
		c.Name = name
		c.ID = truncate(strings.TrimSpace(c.ID), 32)
		categories = append(categories, c)
	}

	replace := r.URL.Query().Get("mode") == "replace"
	added, err := s.store.Import(categories, cleaned, replace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Không thể nhập dữ liệu")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"imported": added, "skipped": skipped})
}

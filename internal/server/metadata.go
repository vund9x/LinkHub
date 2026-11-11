package server

import (
	"context"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var (
	spaceRe       = regexp.MustCompile(`\s+`)
	titleRe       = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	descRe        = regexp.MustCompile(`(?is)<meta[^>]+(?:name|property)\s*=\s*["'](?:description|og:description)["'][^>]*content\s*=\s*["']([^"']*)["']`)
	descReReverse = regexp.MustCompile(`(?is)<meta[^>]+content\s*=\s*["']([^"']*)["'][^>]*(?:name|property)\s*=\s*["'](?:description|og:description)["']`)
)

func (s *Server) handleMetadata(w http.ResponseWriter, r *http.Request) {
	normalized, err := normalizeURL(r.URL.Query().Get("url"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, normalized, nil)
	if err != nil {
		writeError(w, http.StatusBadRequest, "URL không hợp lệ")
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; LinkHub/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := s.client.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "Không thể truy cập trang này")
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	page := string(body)

	title := ""
	if m := titleRe.FindStringSubmatch(page); len(m) > 1 {
		title = cleanText(m[1])
	}
	description := ""
	if m := descRe.FindStringSubmatch(page); len(m) > 1 {
		description = cleanText(m[1])
	} else if m := descReReverse.FindStringSubmatch(page); len(m) > 1 {
		description = cleanText(m[1])
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"title":       truncate(title, 120),
		"description": truncate(description, 240),
	})
}

func cleanText(s string) string {
	s = html.UnescapeString(s)
	s = spaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

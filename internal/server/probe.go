package server

import (
	"context"
	"net/http"
	"sync"
	"time"
)

func (s *Server) handleCheck(w http.ResponseWriter, r *http.Request) {
	var in struct {
		URLs []string `json:"urls"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(in.URLs) > 100 {
		in.URLs = in.URLs[:100]
	}

	type result struct {
		URL    string `json:"url"`
		OK     bool   `json:"ok"`
		Status int    `json:"status,omitempty"`
		Error  string `json:"error,omitempty"`
	}

	results := make([]result, len(in.URLs))
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup

	for i, raw := range in.URLs {
		normalized, err := normalizeURL(raw)
		if err != nil {
			results[i] = result{URL: raw, OK: false, Error: "URL không hợp lệ"}
			continue
		}
		wg.Add(1)
		go func(idx int, target string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			ok, status, errMsg := probe(s.client, target)
			results[idx] = result{URL: target, OK: ok, Status: status, Error: errMsg}
		}(i, normalized)
	}
	wg.Wait()

	writeJSON(w, http.StatusOK, results)
}

func probe(client *http.Client, target string) (bool, int, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, target, nil)
	if err != nil {
		return false, 0, "URL không hợp lệ"
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; LinkHub/1.0)")

	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed && resp.StatusCode != http.StatusNotImplemented {
			return resp.StatusCode < 500, resp.StatusCode, ""
		}
	}

	// Một số server không hỗ trợ HEAD, thử GET với Range rất nhỏ.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	req2, err := http.NewRequestWithContext(ctx2, http.MethodGet, target, nil)
	if err != nil {
		return false, 0, "URL không hợp lệ"
	}
	req2.Header.Set("User-Agent", "Mozilla/5.0 (compatible; LinkHub/1.0)")
	req2.Header.Set("Range", "bytes=0-512")

	resp2, err := client.Do(req2)
	if err != nil {
		return false, 0, "Không kết nối được"
	}
	defer resp2.Body.Close()
	return resp2.StatusCode < 500, resp2.StatusCode, ""
}

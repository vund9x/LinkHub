// Package store quản lý danh sách bookmark và lưu xuống file JSON nguyên tử.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"linkhub/internal/link"
)

type dbFile struct {
	Version int         `json:"version"`
	Links   []link.Link `json:"links"`
}

// Store quản lý danh sách link và ghi xuống file JSON nguyên tử.
type Store struct {
	mu    sync.RWMutex
	path  string
	links []link.Link
}

// New nạp dữ liệu từ path; tạo thư mục và dữ liệu mẫu nếu file chưa tồn tại.
func New(path string) (*Store, error) {
	s := &Store{path: path}

	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		s.links = seedLinks()
		s.mu.Lock()
		err = s.saveLocked()
		s.mu.Unlock()
		return s, err
	case err != nil:
		return nil, err
	}

	if len(data) == 0 {
		s.links = []link.Link{}
		return s, nil
	}

	var db dbFile
	if err := json.Unmarshal(data, &db); err != nil {
		return nil, err
	}
	if db.Links == nil {
		db.Links = []link.Link{}
	}
	s.links = db.Links
	return s, nil
}

func seedLinks() []link.Link {
	now := time.Now().UTC()
	seeds := []struct {
		title, url, desc, category string
		pinned                     bool
	}{
		{"GitHub", "https://github.com", "Mã nguồn, pull request và cộng tác dự án.", "Phát triển", true},
		{"ChatGPT", "https://chatgpt.com", "Trợ lý AI cho công việc hằng ngày.", "Công cụ", false},
		{"YouTube", "https://www.youtube.com", "Video, nhạc và nội dung giải trí.", "Giải trí", false},
		{"Google Drive", "https://drive.google.com", "Tài liệu và tệp dùng chung trên đám mây.", "Công cụ", false},
		{"Cloudflare Speed Test", "https://speed.cloudflare.com", "Đo tốc độ mạng của bạn.", "Tiện ích", false},
	}
	links := make([]link.Link, 0, len(seeds))
	for i, sd := range seeds {
		links = append(links, link.Link{
			ID:          newID(),
			Title:       sd.title,
			URL:         sd.url,
			Description: sd.desc,
			Category:    sd.category,
			Pinned:      sd.pinned,
			CreatedAt:   now,
			Order:       i,
		})
	}
	return links
}

func newID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return time.Now().UTC().Format("150405.000000000")
	}
	return hex.EncodeToString(b)
}

// List trả về bản sao danh sách đã sắp xếp theo thứ tự hiển thị.
func (s *Store) List() []link.Link {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]link.Link, len(s.links))
	copy(out, s.links)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Order != out[j].Order {
			return out[i].Order < out[j].Order
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}

func (s *Store) Create(l link.Link) (link.Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	l.ID = newID()
	l.CreatedAt = time.Now().UTC()
	l.Order = s.nextOrderLocked()
	l.Visits = 0
	l.LastVisited = nil

	s.links = append(s.links, l)
	if err := s.saveLocked(); err != nil {
		s.links = s.links[:len(s.links)-1]
		return link.Link{}, err
	}
	return l, nil
}

// Update áp dụng thay đổi lên link và hoàn tác nếu ghi file thất bại.
func (s *Store) Update(id string, apply func(*link.Link)) (link.Link, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexOfLocked(id)
	if idx < 0 {
		return link.Link{}, false, nil
	}

	old := s.links[idx]
	updated := s.links[idx]
	apply(&updated)
	updated.ID = old.ID
	updated.CreatedAt = old.CreatedAt
	updated.Order = old.Order
	updated.Visits = old.Visits
	updated.LastVisited = old.LastVisited

	s.links[idx] = updated
	if err := s.saveLocked(); err != nil {
		s.links[idx] = old
		return link.Link{}, true, err
	}
	return s.links[idx], true, nil
}

func (s *Store) Delete(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexOfLocked(id)
	if idx < 0 {
		return false, nil
	}

	old := make([]link.Link, len(s.links))
	copy(old, s.links)

	s.links = append(s.links[:idx], s.links[idx+1:]...)
	if err := s.saveLocked(); err != nil {
		s.links = old
		return false, err
	}
	return true, nil
}

func (s *Store) RecordVisit(id string) (link.Link, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexOfLocked(id)
	if idx < 0 {
		return link.Link{}, false, nil
	}

	now := time.Now().UTC()
	s.links[idx].Visits++
	s.links[idx].LastVisited = &now
	if err := s.saveLocked(); err != nil {
		return link.Link{}, true, err
	}
	return s.links[idx], true, nil
}

func (s *Store) Reorder(ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	byID := make(map[string]link.Link, len(s.links))
	for _, l := range s.links {
		byID[l.ID] = l
	}

	next := make([]link.Link, 0, len(s.links))
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if l, ok := byID[id]; ok && !seen[id] {
			next = append(next, l)
			seen[id] = true
		}
	}
	if len(next) != len(s.links) {
		// Giữ lại các link không có trong danh sách gửi lên, theo thứ tự cũ.
		rest := make([]link.Link, len(s.links))
		copy(rest, s.links)
		sort.SliceStable(rest, func(i, j int) bool { return rest[i].Order < rest[j].Order })
		for _, l := range rest {
			if !seen[l.ID] {
				next = append(next, l)
			}
		}
	}
	for i := range next {
		next[i].Order = i
	}

	old := s.links
	s.links = next
	if err := s.saveLocked(); err != nil {
		s.links = old
		return err
	}
	return nil
}

func (s *Store) Import(links []link.Link, replace bool) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if replace {
		s.links = nil
	}

	start := s.nextOrderLocked()
	existing := make(map[string]bool, len(s.links))
	for _, l := range s.links {
		existing[l.ID] = true
	}

	added := 0
	now := time.Now().UTC()
	for _, l := range links {
		if l.ID == "" || existing[l.ID] {
			l.ID = newID()
		}
		existing[l.ID] = true
		if l.CreatedAt.IsZero() {
			l.CreatedAt = now
		}
		l.Order = start + added
		if l.Visits < 0 {
			l.Visits = 0
		}
		s.links = append(s.links, l)
		added++
	}

	if err := s.saveLocked(); err != nil {
		return 0, err
	}
	return added, nil
}

func (s *Store) indexOfLocked(id string) int {
	for i := range s.links {
		if s.links[i].ID == id {
			return i
		}
	}
	return -1
}

func (s *Store) nextOrderLocked() int {
	max := -1
	for _, l := range s.links {
		if l.Order > max {
			max = l.Order
		}
	}
	return max + 1
}

func (s *Store) saveLocked() error {
	db := dbFile{Version: 1, Links: s.links}
	if db.Links == nil {
		db.Links = []link.Link{}
	}
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return err
	}

	// Ghi file tạm rồi đổi tên để tránh mất dữ liệu khi ghi dở.
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

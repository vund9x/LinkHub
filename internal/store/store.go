// Package store quản lý danh sách bookmark, danh mục và lưu xuống file JSON nguyên tử.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"linkhub/internal/link"
)

// ErrCategoryNameTaken trả về khi tên danh mục trùng với danh mục đã có.
var ErrCategoryNameTaken = errors.New("tên danh mục đã tồn tại")

// ImportLink là liên kết trong tệp JSON, hỗ trợ trường category kiểu cũ.
type ImportLink struct {
	link.Link
	Category string `json:"category,omitempty"`
}

type dbFile struct {
	Version    int             `json:"version"`
	Categories []link.Category `json:"categories"`
	Links      []ImportLink    `json:"links"`
}

// Store quản lý danh sách link, danh mục và ghi xuống file JSON nguyên tử.
type Store struct {
	mu         sync.RWMutex
	path       string
	links      []link.Link
	categories []link.Category
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
		s.categories, s.links = seedData()
		s.mu.Lock()
		err = s.saveLocked()
		s.mu.Unlock()
		return s, err
	case err != nil:
		return nil, err
	}

	if len(data) == 0 {
		s.links = []link.Link{}
		s.categories = []link.Category{}
		return s, nil
	}

	var db dbFile
	if err := json.Unmarshal(data, &db); err != nil {
		return nil, err
	}
	s.links = make([]link.Link, len(db.Links))
	for i := range db.Links {
		s.links[i] = db.Links[i].Link
	}
	s.categories = db.Categories

	changed := s.normalize(db.Links)
	if db.Version != 2 {
		changed = true
	}
	if changed {
		s.mu.Lock()
		err = s.saveLocked()
		s.mu.Unlock()
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

// normalize làm sạch danh mục và chuyển các trường category kiểu cũ sang categoryIds.
func (s *Store) normalize(raw []ImportLink) bool {
	changed := false

	catIndex := make(map[string]int)
	idRemap := make(map[string]string)
	cats := make([]link.Category, 0, len(s.categories)+4)

	for _, c := range s.categories {
		name := strings.TrimSpace(c.Name)
		if name == "" {
			changed = true
			continue
		}
		if name != c.Name {
			c.Name = name
			changed = true
		}
		key := nameKey(name)
		if idx, ok := catIndex[key]; ok {
			changed = true
			if c.ID != "" && c.ID != cats[idx].ID {
				idRemap[c.ID] = cats[idx].ID
			}
			continue
		}
		if c.ID == "" {
			c.ID = newID()
			changed = true
		}
		if c.CreatedAt.IsZero() {
			c.CreatedAt = time.Now().UTC()
			changed = true
		}
		if c.Order != len(cats) {
			c.Order = len(cats)
			changed = true
		}
		catIndex[key] = len(cats)
		cats = append(cats, c)
	}

	known := make(map[string]bool, len(cats))
	for _, c := range cats {
		known[c.ID] = true
	}

	ensure := func(name string) string {
		key := nameKey(name)
		if idx, ok := catIndex[key]; ok {
			return cats[idx].ID
		}
		c := link.Category{ID: newID(), Name: name, Order: len(cats), CreatedAt: time.Now().UTC()}
		cats = append(cats, c)
		catIndex[key] = len(cats) - 1
		known[c.ID] = true
		changed = true
		return c.ID
	}

	for i := range raw {
		old := raw[i].Link.CategoryIDs
		ids := make([]string, 0, len(old)+1)
		seen := make(map[string]bool, len(old)+1)
		add := func(id string) {
			if mapped, ok := idRemap[id]; ok {
				id = mapped
				changed = true
			}
			if id == "" || !known[id] || seen[id] {
				return
			}
			seen[id] = true
			ids = append(ids, id)
		}
		for _, id := range old {
			add(id)
		}
		if name := strings.TrimSpace(raw[i].Category); name != "" {
			add(ensure(name))
		}
		if !sameIDs(old, ids) {
			changed = true
		}
		s.links[i].CategoryIDs = ids
	}

	s.categories = cats
	return changed
}

func sameIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func nameKey(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(name), " "))
}

func seedData() ([]link.Category, []link.Link) {
	now := time.Now().UTC()
	names := []string{"Phát triển", "Công cụ", "Giải trí", "Tiện ích"}
	cats := make([]link.Category, len(names))
	byName := make(map[string]string, len(names))
	for i, name := range names {
		cats[i] = link.Category{ID: newID(), Name: name, Order: i, CreatedAt: now}
		byName[name] = cats[i].ID
	}

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
			CategoryIDs: []string{byName[sd.category]},
			Pinned:      sd.pinned,
			CreatedAt:   now,
			Order:       i,
		})
	}
	return cats, links
}

func newID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return time.Now().UTC().Format("150405.000000000")
	}
	return hex.EncodeToString(b)
}

// List trả về bản sao danh sách link đã sắp xếp theo thứ tự hiển thị.
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

// Categories trả về bản sao danh mục đã sắp xếp theo thứ tự hiển thị.
func (s *Store) Categories() []link.Category {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]link.Category, len(s.categories))
	copy(out, s.categories)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Order != out[j].Order {
			return out[i].Order < out[j].Order
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}

// CreateCategory thêm danh mục mới; trả về danh mục đã có nếu tên trùng.
func (s *Store) CreateCategory(name string) (link.Category, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := nameKey(name)
	for _, c := range s.categories {
		if nameKey(c.Name) == key {
			return c, false, nil
		}
	}

	c := link.Category{
		ID:        newID(),
		Name:      name,
		Order:     s.nextCategoryOrderLocked(),
		CreatedAt: time.Now().UTC(),
	}
	s.categories = append(s.categories, c)
	if err := s.saveLocked(); err != nil {
		s.categories = s.categories[:len(s.categories)-1]
		return link.Category{}, false, err
	}
	return c, true, nil
}

// UpdateCategory đổi tên danh mục.
func (s *Store) UpdateCategory(id, name string) (link.Category, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i := range s.categories {
		if s.categories[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return link.Category{}, false, nil
	}

	key := nameKey(name)
	for i, c := range s.categories {
		if i != idx && nameKey(c.Name) == key {
			return link.Category{}, true, ErrCategoryNameTaken
		}
	}

	old := s.categories[idx]
	s.categories[idx].Name = name
	if err := s.saveLocked(); err != nil {
		s.categories[idx] = old
		return link.Category{}, true, err
	}
	return s.categories[idx], true, nil
}

// DeleteCategory xóa danh mục và gỡ nó khỏi mọi liên kết.
func (s *Store) DeleteCategory(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i := range s.categories {
		if s.categories[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false, nil
	}

	oldLinks := make([]link.Link, len(s.links))
	copy(oldLinks, s.links)
	oldCats := make([]link.Category, len(s.categories))
	copy(oldCats, s.categories)

	s.categories = append(s.categories[:idx], s.categories[idx+1:]...)
	for i := range s.links {
		if len(s.links[i].CategoryIDs) == 0 {
			continue
		}
		next := make([]string, 0, len(s.links[i].CategoryIDs))
		for _, cid := range s.links[i].CategoryIDs {
			if cid != id {
				next = append(next, cid)
			}
		}
		s.links[i].CategoryIDs = next
	}

	if err := s.saveLocked(); err != nil {
		s.links = oldLinks
		s.categories = oldCats
		return false, err
	}
	return true, nil
}

func (s *Store) Create(l link.Link) (link.Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	l.ID = newID()
	l.CreatedAt = time.Now().UTC()
	l.Order = s.nextOrderLocked()
	l.Visits = 0
	l.LastVisited = nil
	l.CategoryIDs = s.filterCategoryIDsLocked(l.CategoryIDs)

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
	updated.CategoryIDs = s.filterCategoryIDsLocked(updated.CategoryIDs)

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

// Import thêm hoặc thay thế danh mục và liên kết từ tệp JSON.
func (s *Store) Import(categories []link.Category, links []ImportLink, replace bool) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cats := make([]link.Category, 0, len(s.categories)+len(categories))
	catIndex := make(map[string]int)
	known := make(map[string]bool)
	usedID := make(map[string]bool)

	if !replace {
		for _, c := range s.categories {
			cats = append(cats, c)
			catIndex[nameKey(c.Name)] = len(cats) - 1
			known[c.ID] = true
			usedID[c.ID] = true
		}
	}

	for _, c := range categories {
		name := strings.TrimSpace(c.Name)
		if name == "" {
			continue
		}
		key := nameKey(name)
		if _, ok := catIndex[key]; ok {
			continue
		}
		id := strings.TrimSpace(c.ID)
		if id == "" || usedID[id] {
			id = newID()
		}
		c.ID = id
		c.Name = name
		c.Order = len(cats)
		if c.CreatedAt.IsZero() {
			c.CreatedAt = time.Now().UTC()
		}
		cats = append(cats, c)
		catIndex[key] = len(cats) - 1
		known[id] = true
		usedID[id] = true
	}

	ensure := func(name string) string {
		name = strings.TrimSpace(name)
		key := nameKey(name)
		if idx, ok := catIndex[key]; ok {
			return cats[idx].ID
		}
		c := link.Category{ID: newID(), Name: name, Order: len(cats), CreatedAt: time.Now().UTC()}
		cats = append(cats, c)
		catIndex[key] = len(cats) - 1
		known[c.ID] = true
		usedID[c.ID] = true
		return c.ID
	}

	var next []link.Link
	if !replace {
		next = make([]link.Link, len(s.links), len(s.links)+len(links))
		copy(next, s.links)
	}

	idSet := make(map[string]bool, len(next))
	for _, l := range next {
		idSet[l.ID] = true
	}
	start := 0
	for _, l := range next {
		if l.Order >= start {
			start = l.Order + 1
		}
	}

	added := 0
	now := time.Now().UTC()
	for _, in := range links {
		l := in.Link
		ids := make([]string, 0, len(l.CategoryIDs)+1)
		seen := make(map[string]bool, len(l.CategoryIDs)+1)
		add := func(id string) {
			if id == "" || !known[id] || seen[id] {
				return
			}
			seen[id] = true
			ids = append(ids, id)
		}
		for _, id := range l.CategoryIDs {
			add(strings.TrimSpace(id))
		}
		if name := strings.TrimSpace(in.Category); name != "" {
			add(ensure(name))
		}
		l.CategoryIDs = ids
		if l.ID == "" || idSet[l.ID] {
			l.ID = newID()
		}
		idSet[l.ID] = true
		if l.CreatedAt.IsZero() {
			l.CreatedAt = now
		}
		l.Order = start + added
		if l.Visits < 0 {
			l.Visits = 0
		}
		next = append(next, l)
		added++
	}

	oldLinks, oldCats := s.links, s.categories
	s.links, s.categories = next, cats
	if err := s.saveLocked(); err != nil {
		s.links, s.categories = oldLinks, oldCats
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

func (s *Store) nextCategoryOrderLocked() int {
	max := -1
	for _, c := range s.categories {
		if c.Order > max {
			max = c.Order
		}
	}
	return max + 1
}

func (s *Store) filterCategoryIDsLocked(ids []string) []string {
	out := make([]string, 0, len(ids))
	if len(ids) == 0 {
		return out
	}
	known := make(map[string]bool, len(s.categories))
	for _, c := range s.categories {
		known[c.ID] = true
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id == "" || !known[id] || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func (s *Store) saveLocked() error {
	db := dbFile{
		Version:    2,
		Categories: s.categories,
		Links:      make([]ImportLink, len(s.links)),
	}
	for i, l := range s.links {
		db.Links[i] = ImportLink{Link: l}
	}
	if db.Categories == nil {
		db.Categories = []link.Category{}
	}
	if db.Links == nil {
		db.Links = []ImportLink{}
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

// Package link định nghĩa mô hình bookmark và danh mục dùng chung giữa các tầng.
package link

import "time"

// Link là một bookmark trong dashboard.
type Link struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Description string     `json:"description"`
	CategoryIDs []string   `json:"categoryIds"`
	Icon        string     `json:"icon,omitempty"`
	Pinned      bool       `json:"pinned"`
	Visits      int        `json:"visits"`
	LastVisited *time.Time `json:"lastVisited,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	Order       int        `json:"order"`
}

// Category là một danh mục do người dùng quản lý, dùng chung cho nhiều liên kết.
type Category struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Order     int       `json:"order"`
	CreatedAt time.Time `json:"createdAt"`
}

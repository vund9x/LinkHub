// Package link định nghĩa mô hình bookmark dùng chung giữa các tầng.
package link

import "time"

// Link là một bookmark trong dashboard.
type Link struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Description string     `json:"description"`
	Category    string     `json:"category"`
	Icon        string     `json:"icon,omitempty"`
	Pinned      bool       `json:"pinned"`
	Visits      int        `json:"visits"`
	LastVisited *time.Time `json:"lastVisited,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	Order       int        `json:"order"`
}

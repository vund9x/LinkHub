// Package webui nhúng giao diện SPA vào binary.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed index.html static
var files embed.FS

// Index trả về nội dung trang chính.
func Index() ([]byte, error) {
	return files.ReadFile("index.html")
}

// Static trả về cây tài nguyên tĩnh (được phục vụ dưới /static/).
func Static() (fs.FS, error) {
	return fs.Sub(files, "static")
}

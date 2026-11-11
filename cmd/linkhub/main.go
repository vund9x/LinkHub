package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"linkhub/internal/server"
	"linkhub/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "địa chỉ lắng nghe, ví dụ :8080 hoặc 0.0.0.0:8080")
	dataDir := flag.String("data", "data", "thư mục lưu dữ liệu")
	flag.Parse()

	st, err := store.New(filepath.Join(*dataDir, "links.json"))
	if err != nil {
		log.Fatalf("Không thể khởi tạo dữ liệu: %v", err)
	}

	handler, err := server.New(st).Handler()
	if err != nil {
		log.Fatalf("Không thể khởi tạo server: %v", err)
	}

	srv := &http.Server{
		Addr:              *addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	printAddresses(*addr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server dừng: %v", err)
		}
	case <-ctx.Done():
		log.Println("Đang tắt server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Tắt server gặp lỗi: %v", err)
		}
	}
}

func printAddresses(addr string) {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		port = "8080"
	}
	log.Printf("LinkHub đang chạy tại http://localhost:%s", port)

	ifaces, err := net.Interfaces()
	if err != nil {
		return
	}
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				log.Printf("Trong mạng LAN: http://%s:%s", ipnet.IP.String(), port)
			}
		}
	}
}

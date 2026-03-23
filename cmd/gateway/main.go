package main

import (
	"log"
	"os"

	"github.com/dinhbaokhanh/Final-Project-API-Gateway/internal/app"
	"github.com/dinhbaokhanh/Final-Project-API-Gateway/internal/config"
	"github.com/dinhbaokhanh/Final-Project-API-Gateway/internal/middleware"
)

func main() {
	// Bước 1: Đọc và giải mã rễ file cấu hình JSON của toàn bộ Gateway
	cfg, err := config.Load("gateway.json")
	if err != nil {
		log.Fatalf("Lỗi không thể tải file cấu hình Gateway: %v", err)
	}

	// Khởi chạy kết nối Redis cho cơ chế Blacklist token
	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	middleware.InitRedis(redisAddr)
	middleware.InitJWT()

	// Bước 2: Khởi tạo ứng dụng Gateway (Core App) dựa trên cấu hình đã đọc
	gateway, err := app.New(cfg)
	if err != nil {
		log.Fatalf("Lỗi trong quá trình kết nối Core Gateway: %v", err)
	}

	// Bước 3: Chạy Gateway và bắt đầu lắng nghe các Request từ tầng Frontend
	log.Printf("PTIT Gateway đã khởi động thành công và đang lắng nghe trên cổng :%d", cfg.Port)
	if err := gateway.Run(); err != nil {
		log.Fatalf("Gateway đã dừng hoạt động do xảy ra lỗi: %v", err)
	}
}

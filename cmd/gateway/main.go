package main

import (
	"log"
	"os"

	"github.com/dinhbaokhanh/Final-Project-API-Gateway/internal/app"
	"github.com/dinhbaokhanh/Final-Project-API-Gateway/internal/config"
	"github.com/dinhbaokhanh/Final-Project-API-Gateway/internal/middleware"
	"github.com/joho/godotenv"
)

func main() {
	// Nạp biến môi trường từ .env (thành phần dev) hoặc process env (production)
	_ = godotenv.Load()

	// Khởi động audit logger trước tiên để ghi lại mọi sự kiện ngay từ đầu
	middleware.InitAuditLogger()

	// Đọc cấu hình Gateway từ gateway.json
	cfg, err := config.Load("gateway.json")
	if err != nil {
		log.Fatalf("Không thể tải file cấu hình: %v", err)
	}

	// Kết nối Redis cho cơ chế blacklist token
	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	middleware.InitRedis(redisAddr)

	// Nạp JWT_SECRET vào bộ nhớ (crash nếu thiếu biến này)
	middleware.InitJWT()

	// Khởi tạo và chạy Gateway
	gateway, err := app.New(cfg)
	if err != nil {
		log.Fatalf("Khởi tạo Gateway thất bại: %v", err)
	}

	log.Printf("PTIT Gateway đang lắng nghe tại cổng :%d", cfg.Port)
	if err := gateway.Run(); err != nil {
		log.Fatalf("Gateway dừng hoạt động: %v", err)
	}
}

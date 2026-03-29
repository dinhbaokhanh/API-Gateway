package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dinhbaokhanh/Final-Project-API-Gateway/internal/app"
	"github.com/dinhbaokhanh/Final-Project-API-Gateway/internal/config"
	"github.com/dinhbaokhanh/Final-Project-API-Gateway/internal/middleware"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	middleware.InitAuditLogger()

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

	// Nạp JWT_SECRET vào bộ nhớ — crash nếu thiếu để đảm bảo an toàn
	middleware.InitJWT()

	// Khởi động goroutine dọn dẹp rate limiter để tránh memory leak
	middleware.InitRateLimiter()

	// Khởi tạo Gateway
	gateway, err := app.New(cfg)
	if err != nil {
		log.Fatalf("Khởi tạo Gateway thất bại: %v", err)
	}

	// Chạy Gateway trên goroutine riêng để không block luồng chính
	go func() {
		log.Printf("PTIT Gateway đang lắng nghe tại cổng :%d", cfg.Port)
		if err := gateway.Run(); err != nil {
			log.Printf("Gateway dừng: %v", err)
		}
	}()

	// Lắng nghe tín hiệu hệ thống để thực hiện graceful shutdown
	// SIGINT = Ctrl+C, SIGTERM = Docker stop / Kubernetes kill
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Đang dừng Gateway... Chờ các request hiện tại hoàn thành (tối đa 30s)")

	// Đặt timeout 30 giây để drain request trước khi tắt hoàn toàn
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := gateway.Shutdown(ctx); err != nil {
		log.Fatalf("Dừng Gateway không thành công: %v", err)
	}

	log.Println("Gateway đã dừng hoàn toàn.")
}

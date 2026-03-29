package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/dinhbaokhanh/Final-Project-API-Gateway/internal/config"
	"github.com/dinhbaokhanh/Final-Project-API-Gateway/internal/middleware"
	"github.com/dinhbaokhanh/Final-Project-API-Gateway/internal/routing"
)

// App là lõi trung tâm của API Gateway
type App struct {
	server *http.Server
}

// New khởi tạo Gateway từ file cấu hình JSON
func New(cfg *config.GatewayConfig) (*App, error) {
	router, err := routing.NewRouter(cfg)
	if err != nil {
		return nil, err
	}

	// RequestValidation -> AuditLogger -> Recoverer -> RequestLogger -> CORS -> Router
	handler := middleware.Chain(
		router,
		middleware.CORS,
		middleware.RequestLogger,
		middleware.Recoverer,
		middleware.AuditLoggerMiddleware,
		middleware.RequestValidationMiddleware,
	)

	return &App{
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Port),
			Handler:           handler,
			ReadHeaderTimeout: 15 * time.Second, // Tối đa 15s đọc Header
			IdleTimeout:       120 * time.Second, // Giữ kết nối keep-alive tối đa 120s
		},
	}, nil
}

// Run bắt đầu lắng nghe và xử lý request
func (a *App) Run() error {
	return a.server.ListenAndServe()
}

// Shutdown dừng Gateway một cách nhẹ nhàng — cho phép các request hiện tại hoàn thành
func (a *App) Shutdown(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}

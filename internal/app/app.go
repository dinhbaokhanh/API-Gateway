package app

import (
	"fmt"
	"net/http"

	"github.com/dinhbaokhanh/Final-Project-API-Gateway/internal/config"
	"github.com/dinhbaokhanh/Final-Project-API-Gateway/internal/middleware"
	"github.com/dinhbaokhanh/Final-Project-API-Gateway/internal/routing"
)

// App là lõi trung tâm của API Gateway, chứa toàn bộ HTTP Server
type App struct {
	server *http.Server
}

// Khởi tạo cấu trúc và quy trình của Gateway
func New(cfg *config.GatewayConfig) (*App, error) {
	// 1. Tạo Dynamic Router từ cấu hình JSON
	router, err := routing.NewRouter(cfg)
	if err != nil {
		return nil, err
	}

	// 2. Bọc Router qua một chuỗi các Middleware (Phần mềm trung gian)
	handler := middleware.Chain(
		router,
		middleware.Recoverer,
		middleware.RequestLogger,
		middleware.CORS,
	)

	// 3. Khởi tạo HTTP Server trên cổng được chỉ định ở file config
	return &App{
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Port),
			Handler: handler,
		},
	}, nil
}

func (a *App) Run() error {
	return a.server.ListenAndServe()
}

package app

import (
	"fmt"
	"net/http"

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
	// Tạo dynamic router từ cấu hình đã đọc
	router, err := routing.NewRouter(cfg)
	if err != nil {
		return nil, err
	}

	// Chuỗi middleware toàn cục — áp dụng cho mọi request theo thứ tự từ ngoài vào trong:
	// RequestValidation (kiểm tra body/content-type)
	//   -> AuditLogger (ghi log bảo mật)
	//     -> Recoverer (chống crash)
	//       -> RequestLogger (ghi log latency)
	//         -> CORS
	//           -> Router (định tuyến per-route)
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
			Addr:    fmt.Sprintf(":%d", cfg.Port),
			Handler: handler,
		},
	}, nil
}

// Run bắt đầu lắng nghe và xử lý request
func (a *App) Run() error {
	return a.server.ListenAndServe()
}

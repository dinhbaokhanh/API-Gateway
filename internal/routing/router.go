package routing

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/dinhbaokhanh/Final-Project-API-Gateway/internal/config"
	"github.com/dinhbaokhanh/Final-Project-API-Gateway/internal/middleware"
	"github.com/dinhbaokhanh/Final-Project-API-Gateway/internal/proxy"
)

// NewRouter xây dựng HTTP handler với đầy đủ middleware per-route từ cấu hình JSON
func NewRouter(cfg *config.GatewayConfig) (http.Handler, error) {
	mux := http.NewServeMux()

	// Route kiểm tra trạng thái Gateway
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Tạo factory middleware JWT dùng chung cho tất cả route cần xác thực
	authMiddleware := middleware.AuthMiddlewareProvider(cfg.JWT)

	for _, endpoint := range cfg.Endpoints {
		if len(endpoint.Backend) == 0 || len(endpoint.Backend[0].Host) == 0 {
			continue
		}

		targetURL := endpoint.Backend[0].Host[0]

		reverseProxy, err := proxy.NewReverseProxy(targetURL)
		if err != nil {
			return nil, fmt.Errorf("URL backend không hợp lệ cho endpoint %s: %w", endpoint.Endpoint, err)
		}

		// Tạo pattern routing theo chuẩn Go 1.22+: "METHOD /path"
		pattern := endpoint.Endpoint
		if endpoint.Method != "" && endpoint.Method != "ANY" {
			pattern = fmt.Sprintf("%s %s", strings.ToUpper(endpoint.Method), endpoint.Endpoint)
		}

		fmt.Printf("[Router] %-35s -> %s\n", pattern, targetURL)

		// Xây chuỗi handler từ trong ra ngoài:
		// reverseProxy -> (JWT Auth nếu cần) -> Xóa header giả mạo -> RateLimit
		var handler http.Handler = reverseProxy

		// Tầng 1 — Xác thực JWT (chỉ áp dụng nếu route yêu cầu)
		if endpoint.AuthRequired {
			handler = authMiddleware(handler)
		}

		// Tầng 2 — Xóa header định danh người dùng do client tự chèn vào
		inner := handler
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Del("X-User-ID")
			r.Header.Del("X-User-Role")
			inner.ServeHTTP(w, r)
		})

		// Tầng 3 — Rate limiting theo IP
		handler = middleware.RateLimitMiddleware(handler)

		mux.Handle(pattern, handler)
	}

	return mux, nil
}

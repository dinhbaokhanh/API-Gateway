package middleware

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Middleware định nghĩa kiểu hàm bọc HTTP Handler
type Middleware func(http.Handler) http.Handler

// Chain gom nhiều middleware thành một chuỗi xử lý, thứ tự từ ngoài vào trong
func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	wrapped := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrapped = middlewares[i](wrapped)
	}
	return wrapped
}

// RequestLogger ghi log method, đường dẫn và thời gian xử lý của mỗi request
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// Recoverer bắt mọi lỗi panic bên trong, ghi log và trả về 500 thay vì để Gateway sập
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[PANIC] Phục hồi từ lỗi nghiêm trọng: %v", rec)
				http.Error(w, "Lỗi máy chủ nội bộ", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// CORS thiết lập header theo Whitelist được cho phép từ `.env`
func CORS(next http.Handler) http.Handler {
	// Nạp danh sách từ cấu hình môi trường (Ví dụ: "http://localhost:3000, https://soc.ptit.edu.vn")
	allowedOriginsStr := os.Getenv("CORS_ALLOWED_ORIGINS")
	var allowedOrigins []string
	if allowedOriginsStr != "" {
		parts := strings.Split(allowedOriginsStr, ",")
		for _, o := range parts {
			allowedOrigins = append(allowedOrigins, strings.TrimSpace(o))
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowOrigin := ""

		if len(allowedOrigins) == 0 {
			allowOrigin = "*" // Fallback bảo vệ hệ thống không bị panic nếu chưa cấu hình
		} else {
			// Duyệt qua Whitelist để cấp quyền cho đúng domain gọi tới
			for _, o := range allowedOrigins {
				if o == "*" || o == origin {
					allowOrigin = origin
					break
				}
			}
		}

		if allowOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		
		// Kích hoạt Credentials (Cookie-based auth) khi được chỉ định đích danh (không phải wildcard *)
		if allowOrigin != "" && allowOrigin != "*" {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Trình duyệt gửi OPTIONS để kiểm tra trước (preflight)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

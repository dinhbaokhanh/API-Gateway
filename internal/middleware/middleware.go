package middleware

import (
	"log"
	"net/http"
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

// CORS thiết lập header cho phép frontend gọi API từ domain khác (cross-origin)
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")

		// Trình duyệt gửi OPTIONS để kiểm tra trước (preflight), phản hồi ngay không cần xử lý tiếp
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

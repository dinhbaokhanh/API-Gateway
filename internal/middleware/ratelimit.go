package middleware

import (
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// IP-based Rate Limiter (20 req / giây tiêu chuẩn)
var (
	visitors = make(map[string]*rate.Limiter)
	mu       sync.Mutex
)

// getVisitor trả về hoặc tạo mới Limiter cho một IP
func getVisitor(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	limiter, exists := visitors[ip]
	if !exists {
		// rate.Limit(20) cho phép 20 tokens (requests) mỗi giây, b là kích thước bucket = 20
		limiter = rate.NewLimiter(rate.Limit(20), 20)
		visitors[ip] = limiter
	}

	return limiter
}

// RateLimitMiddleware giới hạn số lượng request bằng golang.org/x/time/rate
func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Lấy IP của Client
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			// Nếu split lỗi (như dùng proxy hoặc load balancer sai cấu hình lấy RemoteAddr), dùng toàn bộ RemoteAddr
			ip = r.RemoteAddr
		}

		limiter := getVisitor(ip)
		
		// Kéo token, nếu thất bại nghĩa là vượt ngưỡng 20 req/s
		if !limiter.Allow() {
			http.Error(w, "Too Many Requests - Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

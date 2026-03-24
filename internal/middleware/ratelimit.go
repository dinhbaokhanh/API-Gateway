package middleware

import (
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

var (
	visitors = make(map[string]*rate.Limiter)
	mu       sync.Mutex
)

// getVisitor trả về rate limiter theo IP, tạo mới nếu chưa có
func getVisitor(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	limiter, exists := visitors[ip]
	if !exists {
		// Cho phép tối đa 20 request/giây mỗi IP, burst tối đa 20
		limiter = rate.NewLimiter(rate.Limit(20), 20)
		visitors[ip] = limiter
	}
	return limiter
}

// RateLimitMiddleware chặn request khi IP vượt quá 20 req/giây
func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		if !getVisitor(ip).Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

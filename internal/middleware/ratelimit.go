package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var visitors sync.Map

func InitRateLimiter() {
	go cleanupVisitors()
}

// cleanupVisitors xóa các IP không còn gửi request trong vòng 10 phút
func cleanupVisitors() {
	for {
		time.Sleep(5 * time.Minute)

		visitors.Range(func(key, value interface{}) bool {
			v := value.(*visitor)
			if time.Since(v.lastSeen) > 10*time.Minute {
				visitors.Delete(key)
			}
			return true
		})
	}
}

// Trả về rate limiter của một IP
func getVisitor(ip string) *rate.Limiter {
	// Lấy hoặc khởi tạo Limiter mới
	vInfo, loaded := visitors.LoadOrStore(ip, &visitor{
		limiter:  rate.NewLimiter(rate.Limit(20), 20),
		lastSeen: time.Now(),
	})

	v := vInfo.(*visitor)
	if loaded {
		v.lastSeen = time.Now()
	}

	return v.limiter
}

// RateLimitMiddleware chặn request spam
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

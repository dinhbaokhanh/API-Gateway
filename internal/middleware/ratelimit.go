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

// Dùng sync.Map để xử lý truy cập đồng thời triệt để, khắc phục Race Condition
var visitors sync.Map

// InitRateLimiter khởi động worker dọn dẹp bộ nhớ mỗi 5 phút.
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
			return true // Tiếp tục vòng lặp
		})
	}
}

// getVisitor trả về rate limiter của một IP
func getVisitor(ip string) *rate.Limiter {
	// Lấy hoặc khởi tạo Limiter mới (tối đa 20 req/s, burst 20)
	vInfo, loaded := visitors.LoadOrStore(ip, &visitor{
		limiter:  rate.NewLimiter(rate.Limit(20), 20),
		lastSeen: time.Now(),
	})

	v := vInfo.(*visitor)
	if loaded {
		// Dù đã có sẵn, ta vẫn cập nhật lại thời gian lastSeen
		v.lastSeen = time.Now()
	}

	return v.limiter
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

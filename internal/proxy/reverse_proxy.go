package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/sony/gobreaker/v2"
)

// RoundRobinProxy phân tải HTTP traffic lần lượt qua các backend để tăng năng lực phục vụ
type RoundRobinProxy struct {
	proxies []*httputil.ReverseProxy
	current uint32
}

func (rr *RoundRobinProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(rr.proxies) == 0 {
		http.Error(w, "No backends available", http.StatusBadGateway)
		return
	}
	// Xoay vòng index backend bằng thuật toán tịnh tiến nguyên tử vòng lặp (Atomic round-robin)
	idx := atomic.AddUint32(&rr.current, 1) % uint32(len(rr.proxies))
	rr.proxies[idx].ServeHTTP(w, r)
}

// NewLoadBalancedProxy tạo proxy phân tải đến danh sách nhiều server backend
func NewLoadBalancedProxy(targets []string, timeoutSec int) (http.Handler, error) {
	if len(targets) == 0 {
		return nil, fmt.Errorf("không có backend nào được cấp")
	}

	if timeoutSec <= 0 {
		timeoutSec = 15 // Mặc định 15 giây nếu chưa được config
	}
	timeout := time.Duration(timeoutSec) * time.Second

	// Dùng chung cấu hình Transport cho tất cả proxy để tối ưu Connection Pool
	transport := &http.Transport{
		ResponseHeaderTimeout: timeout,
		MaxIdleConns:          5000,
		MaxIdleConnsPerHost:   1000,
		IdleConnTimeout:       120 * time.Second,
	}

	proxies := make([]*httputil.ReverseProxy, 0, len(targets))

	for _, target := range targets {
		targetURL, err := url.Parse(target)
		if err != nil {
			return nil, err
		}

		p := httputil.NewSingleHostReverseProxy(targetURL)

		// Mỗi target có một Circuit Breaker độc lập (bảo vệ Host đó)
		cb := gobreaker.NewCircuitBreaker[*http.Response](gobreaker.Settings{
			Name:        "CB-" + targetURL.Host,
			MaxRequests: 5,                  // Số request cho phép thử qua khi Half-Open
			Interval:    10 * time.Second,   // Thời gian đếm lỗi để reset counter vòng lặp
			Timeout:     15 * time.Second,   // Thời gian Open (15s sẽ chuyển về Half-Open)
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
				// Cầu dao sập nếu lượng request >= 10 và tỉ lệ rớt >= 50%
				return counts.Requests >= 10 && failureRatio >= 0.5
			},
		})

		p.Transport = &breakerTransport{
			cb:        cb,
			transport: transport,
		}

		// Ghi đè Director để sửa lỗi header Host
		originalDirector := p.Director
		p.Director = func(req *http.Request) {
			originalDirector(req)
			req.Host = targetURL.Host
		}

		p.ErrorHandler = func(w http.ResponseWriter, r *http.Request, proxyErr error) {
			http.Error(w, "Dịch vụ backend hiện không khả dụng do lỗi kết nối", http.StatusBadGateway)
		}

		proxies = append(proxies, p)
	}

	// Tối ưu: Nếu chỉ khai báo 1 server trong gateway.json, không cần tải tầng Load Balancer array
	if len(proxies) == 1 {
		return proxies[0], nil
	}

	// Trả về Load Balancer cho nhiều server
	return &RoundRobinProxy{proxies: proxies}, nil
}

// breakerTransport bọc http.RoundTripper qua cơ chế Circuit Breaker
type breakerTransport struct {
	cb        *gobreaker.CircuitBreaker[*http.Response]
	transport http.RoundTripper
}

func (b *breakerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Execute chạy code HTTP call và theo dõi kết quả trả về
	resp, err := b.cb.Execute(func() (*http.Response, error) {
		return b.transport.RoundTrip(req)
	})
	if err != nil {
		// Nếu CB Open, err sẽ tự văng ra, ReverseProxy nhận được err và đẩy vào ErrorHandler
		return nil, err
	}
	return resp, nil
}

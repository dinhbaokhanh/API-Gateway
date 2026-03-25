package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// NewReverseProxy tạo reverse proxy đến backend đích với cấu hình timeout động
func NewReverseProxy(target string, timeoutSec int) (http.Handler, error) {
	if timeoutSec <= 0 {
		timeoutSec = 15 // Mặc định 15 giây nếu chưa được config
	}
	timeout := time.Duration(timeoutSec) * time.Second

	targetURL, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	p := httputil.NewSingleHostReverseProxy(targetURL)

	// Gắn http.Client có timeout để tránh backend chậm/treo làm nghẽn Gateway
	p.Transport = &http.Transport{
		ResponseHeaderTimeout: timeout,          // Timeout chờ phản hồi header từ backend
		MaxIdleConns:          100,              // Tối đa 100 kết nối idle trong pool
		MaxIdleConnsPerHost:   10,               // Tối đa 10 idle connection per backend host
		IdleConnTimeout:       90 * time.Second,
	}

	// Ghi đè Director để fix header Host — nếu không Go gửi "localhost:8080" thay vì host thật
	originalDirector := p.Director
	p.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = targetURL.Host
	}

	// Trả lỗi 502 thay vì để lộ thông báo lỗi kỹ thuật ra ngoài
	p.ErrorHandler = func(w http.ResponseWriter, r *http.Request, proxyErr error) {
		http.Error(w, "Dịch vụ backend hiện không khả dụng", http.StatusBadGateway)
	}

	return p, nil
}

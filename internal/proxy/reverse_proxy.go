package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"
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
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
	}

	proxies := make([]*httputil.ReverseProxy, 0, len(targets))

	for _, target := range targets {
		targetURL, err := url.Parse(target)
		if err != nil {
			return nil, err
		}

		p := httputil.NewSingleHostReverseProxy(targetURL)
		p.Transport = transport

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

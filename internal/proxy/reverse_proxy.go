package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// NewReverseProxy tạo mới một Reverse Proxy trỏ tới một backend đích
func NewReverseProxy(target string) (http.Handler, error) {
	targetURL, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	p := httputil.NewSingleHostReverseProxy(targetURL)

	// Lấy Director gốc của SingleHostReverseProxy
	originalDirector := p.Director
	
	// Thay thế bằng Director tùy chỉnh
	p.Director = func(req *http.Request) {
		// Chạy logic chuẩn (Set scheme, path...)
		originalDirector(req)
		
		// QUAN TRỌNG: Ghi đè header "Host" của request bằng Host của server đích (VD: chatbox-server.onrender.com)
		// Nếu không có dòng này, Go sẽ lấy Host gốc là localhost:8080 gửi đi
		req.Host = targetURL.Host
	}

	p.ErrorHandler = func(w http.ResponseWriter, r *http.Request, proxyErr error) {
		http.Error(w, "Lỗi từ Backend: Dịch vụ hiện không khả dụng", http.StatusBadGateway)
	}

	return p, nil
}

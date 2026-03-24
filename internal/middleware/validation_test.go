package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// handler giả để kiểm tra request đi qua middleware thành công
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func TestRequestValidation_OK(t *testing.T) {
	// Request POST với Content-Type và body hợp lệ
	body := bytes.NewBufferString(`{"key": "value"}`)
	req := httptest.NewRequest("POST", "/api/test", body)
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(body.Len())

	rr := httptest.NewRecorder()
	RequestValidationMiddleware(okHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Mong đợi 200, nhận %d", rr.Code)
	}
}

func TestRequestValidation_UnsupportedContentType(t *testing.T) {
	// Content-Type không nằm trong danh sách cho phép
	body := bytes.NewBufferString("some data")
	req := httptest.NewRequest("POST", "/api/test", body)
	req.Header.Set("Content-Type", "text/plain")
	req.ContentLength = int64(body.Len())

	rr := httptest.NewRecorder()
	RequestValidationMiddleware(okHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Errorf("Mong đợi 415, nhận %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "unsupported_media_type") {
		t.Errorf("Body phải chứa 'unsupported_media_type', nhận: %s", rr.Body.String())
	}
}

func TestRequestValidation_PayloadTooLarge(t *testing.T) {
	// Tạo body >1MB để kiểm tra giới hạn kích thước
	oversized := bytes.Repeat([]byte("a"), (1<<20)+1)
	req := httptest.NewRequest("POST", "/api/test", bytes.NewReader(oversized))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len(oversized))

	rr := httptest.NewRecorder()
	// Dùng một handler thực sự đọc body để trigger lỗi MaxBytesReader
	readBodyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 2<<20)
		_, err := r.Body.Read(buf)
		if err != nil && err.Error() == "http: request body too large" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	RequestValidationMiddleware(readBodyHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Mong đợi 413, nhận %d", rr.Code)
	}
}

func TestRequestValidation_NoContentTypeOK(t *testing.T) {
	// GET request không có body — phải được cho qua bình thường
	req := httptest.NewRequest("GET", "/api/test", nil)

	rr := httptest.NewRecorder()
	RequestValidationMiddleware(okHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Mong đợi 200 cho GET không có body, nhận %d", rr.Code)
	}
}

func TestRequestValidation_MultipartOK(t *testing.T) {
	// multipart/form-data phải được chấp nhận
	body := bytes.NewBufferString("--boundary\r\nContent-Disposition: form-data; name=\"field\"\r\n\r\nvalue\r\n--boundary--")
	req := httptest.NewRequest("POST", "/api/upload", body)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	req.ContentLength = int64(body.Len())

	rr := httptest.NewRecorder()
	RequestValidationMiddleware(okHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Mong đợi 200 cho multipart, nhận %d", rr.Code)
	}
}

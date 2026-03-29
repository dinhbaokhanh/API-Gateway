package middleware

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
)

const maxBodyBytes = 1 << 20 // 1MB

var allowedContentTypes = []string{
	"application/json",
	"application/x-www-form-urlencoded",
	"multipart/form-data",
}

// RequestValidationMiddleware kiểm tra kích thước và định dạng của request đến.
func RequestValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Nhận diện request có mang payload: ContentLength > 0 hoặc dùng Chunked Encoding
		hasBody := r.ContentLength > 0 || len(r.TransferEncoding) > 0

		// Bọc body bằng MaxBytesReader để tự động từ chối nếu vượt giới hạn (áp dụng cả với chunked)
		if hasBody {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
			ct := r.Header.Get("Content-Type")

			if ct == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnsupportedMediaType)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "missing_content_type",
				})
				return
			}

			mediaType := strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))
			if !isAllowedContentType(mediaType) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnsupportedMediaType)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "unsupported_media_type",
				})
				return
			}
		}

		// Bắt lỗi vượt kích thước sau khi middleware tiếp theo đọc body
		next.ServeHTTP(w, r)

		// Kiểm tra xem lỗi 413 có bị trigger bởi MaxBytesReader không
		if r.Body != nil {
			buf := make([]byte, 1)
			_, readErr := r.Body.Read(buf)
			if readErr != nil && readErr.Error() == "http: request body too large" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "payload_too_large",
				})
				return
			}
		}
	})
}

// isAllowedContentType kiểm tra xem media type có nằm trong danh sách cho phép không
func isAllowedContentType(mediaType string) bool {
	return slices.Contains(allowedContentTypes, mediaType)
}

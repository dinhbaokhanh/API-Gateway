package middleware

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

// SecurityEvent là cấu trúc đại diện cho một sự kiện bảo mật cần được ghi lại.
// Mỗi sự kiện đầu ra là một dòng JSON trên stdout để dễ dàng đọc bởi hệ thống logging (ELK, Loki, etc.)
type SecurityEvent struct {
	Timestamp  time.Time `json:"ts"`
	IP         string    `json:"ip"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	StatusCode int       `json:"status_code"`
	Reason     string    `json:"reason"`
	JTI        string    `json:"jti,omitempty"`
	UserID     string    `json:"user_id,omitempty"`
	UserRole   string    `json:"user_role,omitempty"`
}

// Các hằng số lý do ghi log bảo mật — dùng chung trong toàn bộ hệ thống
const (
	ReasonRateLimited          = "rate_limited"
	ReasonInvalidJWT           = "invalid_jwt"
	ReasonBlacklistedToken     = "blacklisted_token"
	ReasonForbidden            = "forbidden"
	ReasonPayloadTooLarge      = "payload_too_large"
	ReasonUnsupportedMediaType = "unsupported_media_type"
	ReasonAuthOK               = "auth_ok"
)

// auditLogger là instance logger duy nhất, được khởi tạo một lần khi startup
var auditLogger *log.Logger

// InitAuditLogger khởi tạo logger ghi log bảo mật ra stdout dạng JSON thuần
func InitAuditLogger() {
	// Dùng log.New để tắt prefix mặc định (ngày giờ) vì chúng ta tự ghi trong SecurityEvent
	auditLogger = log.New(os.Stdout, "", 0)
}

// LogSecurityEvent ghi một sự kiện bảo mật ra stdout dạng JSON trên một dòng
func LogSecurityEvent(event SecurityEvent) {
	if auditLogger == nil {
		return
	}
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	auditLogger.Println(string(data))
}

// responseWriter là wrapper xung quanh http.ResponseWriter để ghi lại status code
// vì Go không cung cấp cách đọc lại status code sau khi đã WriteHeader
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK} // Mặc định là 200
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// AuditLoggerMiddleware bọc toàn bộ chuỗi middleware, ghi lại kết quả cuối cùng của mỗi request.
// Phải được đặt ở vị trí ngoài cùng để bắt được mọi từ chối bên trong.
func AuditLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Lấy IP thực của client
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		// Bọc ResponseWriter để theo dõi status code được ghi ra
		wrapped := newResponseWriter(w)
		next.ServeHTTP(wrapped, r)

		// Sau khi toàn bộ chuỗi xử lý xong, ghi log bảo mật nếu là sự kiện đáng chú ý
		statusCode := wrapped.statusCode
		if statusCode == http.StatusUnauthorized ||
			statusCode == http.StatusForbidden ||
			statusCode == http.StatusTooManyRequests ||
			statusCode == http.StatusRequestEntityTooLarge ||
			statusCode == http.StatusUnsupportedMediaType {

			// Xác định lý do bị từ chối dựa trên status code
			reason := inferReason(statusCode, r)

			LogSecurityEvent(SecurityEvent{
				Timestamp:  time.Now().UTC(),
				IP:         ip,
				Method:     r.Method,
				Path:       r.URL.Path,
				StatusCode: statusCode,
				Reason:     reason,
				UserID:     r.Header.Get("X-User-ID"),
				UserRole:   r.Header.Get("X-User-Role"),
			})
		}
	})
}

// inferReason suy ra lý do từ chối theo status code của response
func inferReason(statusCode int, r *http.Request) string {
	switch statusCode {
	case http.StatusTooManyRequests:
		return ReasonRateLimited
	case http.StatusUnauthorized:
		return ReasonInvalidJWT
	case http.StatusForbidden:
		return ReasonForbidden
	case http.StatusRequestEntityTooLarge:
		return ReasonPayloadTooLarge
	case http.StatusUnsupportedMediaType:
		return ReasonUnsupportedMediaType
	default:
		return "unknown"
	}
}

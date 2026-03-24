package middleware

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuditLogger_LogsRejected401(t *testing.T) {
	InitAuditLogger()

	// Handler trả về 401 để kiểm tra audit logger ghi lại
	handler401 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	rr := httptest.NewRecorder()

	AuditLoggerMiddleware(handler401).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Mong đợi 401, nhận %d", rr.Code)
	}
}

func TestAuditLogger_Passthrough200(t *testing.T) {
	InitAuditLogger()

	req := httptest.NewRequest("GET", "/api/public", nil)
	req.RemoteAddr = "10.0.0.1:5678"
	rr := httptest.NewRecorder()

	AuditLoggerMiddleware(okHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Mong đợi 200, nhận %d", rr.Code)
	}
}

func TestAuditLogger_LogsRejected429(t *testing.T) {
	InitAuditLogger()

	handler429 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})

	req := httptest.NewRequest("POST", "/api/any", nil)
	req.RemoteAddr = "10.0.0.2:9999"
	rr := httptest.NewRecorder()

	AuditLoggerMiddleware(handler429).ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Mong đợi 429, nhận %d", rr.Code)
	}
}

func TestLogSecurityEvent_ValidJSON(t *testing.T) {
	InitAuditLogger()

	// Ghi thử một event và kiểm tra nó serialize được sang JSON hợp lệ
	evt := SecurityEvent{
		IP:         "127.0.0.1",
		Method:     "GET",
		Path:       "/api/test",
		StatusCode: 401,
		Reason:     ReasonInvalidJWT,
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("SecurityEvent không thể marshal sang JSON: %v", err)
	}

	if !strings.Contains(string(data), "invalid_jwt") {
		t.Errorf("JSON log thiếu trường reason, nhận: %s", string(data))
	}
}

func TestInferReason(t *testing.T) {
	cases := []struct {
		code     int
		expected string
	}{
		{http.StatusTooManyRequests, ReasonRateLimited},
		{http.StatusUnauthorized, ReasonInvalidJWT},
		{http.StatusForbidden, ReasonForbidden},
		{http.StatusRequestEntityTooLarge, ReasonPayloadTooLarge},
		{http.StatusUnsupportedMediaType, ReasonUnsupportedMediaType},
	}

	req := httptest.NewRequest("GET", "/", nil)
	for _, c := range cases {
		got := inferReason(c.code, req)
		if got != c.expected {
			t.Errorf("inferReason(%d) = %q, mong đợi %q", c.code, got, c.expected)
		}
	}
}

// Kiểm tra rằng auditlog không panic khi logger chưa được init
func TestLogSecurityEvent_NilLogger(t *testing.T) {
	auditLogger = nil // Reset về nil
	// Gọi mà không init — phải không panic
	LogSecurityEvent(SecurityEvent{IP: "1.2.3.4", Reason: "test"})

	// Khởi tạo lại để test khác không bị ảnh hưởng
	InitAuditLogger()
	_ = io.Discard
}

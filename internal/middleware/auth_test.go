package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/dinhbaokhanh/Final-Project-API-Gateway/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

func TestAuthMiddleware(t *testing.T) {
	// Setup mô phỏng Redis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Không thể khởi động miniredis: %v", err)
	}
	defer mr.Close()
	InitRedis(mr.Addr())

	// Setup JWT config
	os.Setenv("JWT_SECRET", "supersecretkey")
	InitJWT()
	cfg := config.JWTConfig{
		Issuer:   "test-issuer",
		Audience: "test-audience",
	}
	authMw := AuthMiddlewareProvider(cfg)

	// Handler giả mô phỏng Backend sau khi qua Gateway
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(r.Header.Get("X-User-ID")))
	})

	tests := []struct {
		name           string
		setupRequest   func() *http.Request
		expectedStatus int
		expectedHeader string
	}{
		{
			name: "Không có Header",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/api/test", nil)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Header sai định dạng",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/api/test", nil)
				req.Header.Set("Authorization", "Token abcxyz")
				return req
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Token hợp lệ, Client giả mạo Header ID",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/api/test", nil)
				// Client cố tình gắn header X-User-ID là admin
				req.Header.Set("X-User-ID", "admin999")
				
				// Sinh token thực sự chứa jti, exp, iss, aud và id là 123
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"jti":  "my-session-id",
					"exp":  time.Now().Add(time.Hour).Unix(),
					"iss":  "test-issuer",
					"aud":  "test-audience",
					"id":   "real123",
					"role": "user",
				})
				tokenString, _ := token.SignedString([]byte("supersecretkey"))
				req.Header.Set("Authorization", "Bearer "+tokenString)
				return req
			},
			expectedStatus: http.StatusOK,
			expectedHeader: "real123", // Fake header must be overridden by token claims!
		},
		{
			name: "Token hết hạn",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/api/test", nil)
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"jti": "my-session-id",
					"exp": time.Now().Add(-time.Hour).Unix(), // Quá khứ
					"iss": "test-issuer",
					"aud": "test-audience",
				})
				tokenString, _ := token.SignedString([]byte("supersecretkey"))
				req.Header.Set("Authorization", "Bearer "+tokenString)
				return req
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Token bị từ chối do sai Issuer",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/api/test", nil)
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"jti": "my-session-id",
					"exp": time.Now().Add(time.Hour).Unix(),
					"iss": "hacker-issuer", // Sai issuer
					"aud": "test-audience",
				})
				tokenString, _ := token.SignedString([]byte("supersecretkey"))
				req.Header.Set("Authorization", "Bearer "+tokenString)
				return req
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			rr := httptest.NewRecorder()

			// Chạy request qua middleware
			handler := authMw(nextHandler)
			
			// Thêm lớp strip header giống như router!
			finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r.Header.Del("X-User-ID")
				r.Header.Del("X-User-Role")
				handler.ServeHTTP(w, r)
			})

			finalHandler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler %s trả về %v, mong đợi %v. Body: %s", tt.name, status, tt.expectedStatus, rr.Body.String())
			}

			if tt.expectedStatus == http.StatusOK {
				// Body của nextHandler trả lại giá trị X-User-ID
				if rr.Body.String() != tt.expectedHeader {
					t.Errorf("Mong đợi X-User-ID là %v, nhận được %v", tt.expectedHeader, rr.Body.String())
				}
			}
		})
	}
}

func TestAlgNoneBlocked(t *testing.T) {
	os.Setenv("JWT_SECRET", "supersecretkey")
	InitJWT()
	cfg := config.JWTConfig{Issuer: "i", Audience: "a"}
	authMw := AuthMiddlewareProvider(cfg)

	req := httptest.NewRequest("GET", "/api/test", nil)
	// Tạo token alg=none 
	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"jti": "jti1",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iss": "i",
		"aud": "a",
	})
	tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	rr := httptest.NewRecorder()
	authMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Mong đợi bị block (401) nhưng nhận %v khi truyền token alg:none", rr.Code)
	}
}

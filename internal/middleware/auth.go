package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/dinhbaokhanh/Final-Project-API-Gateway/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret []byte

// InitJWT thiết lập JWT Secret key. Sẽ crash (panic) nếu không cấu hình!
func InitJWT() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Yêu cầu bắt buộc từ User: Crash lúc khởi động nếu không có JWT_SECRET
		panic("CRITICAL: Biến môi trường JWT_SECRET bị thiếu. Hãy cấu hình để Gateway chạy an toàn!")
	}
	jwtSecret = []byte(secret)
}

// AuthMiddlewareProvider trả về một HTTP middleware hoàn thiện, dùng cấu hình (Issuer, Audience) từ gateway.json
func AuthMiddlewareProvider(jwtCfg config.JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			
			// 1. Phân tích header Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Unauthorized - Missing or invalid Authorization header", http.StatusUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			// 2. Parse và xác minh Token chống alg:none attack và đảm bảo 4 claims (exp, iss, aud, jti)
			token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				// Ép kiểu xác nhận thuật toán HMAC
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}
				return jwtSecret, nil
			}, jwt.WithValidMethods([]string{"HS256"}),
				jwt.WithExpirationRequired(),
				jwt.WithIssuer(jwtCfg.Issuer),
				jwt.WithAudience(jwtCfg.Audience))

			if err != nil || !token.Valid {
				http.Error(w, "Unauthorized - Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// 3. Trích xuất Payload (Claims)
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, "Unauthorized - Failed to read claims", http.StatusUnauthorized)
				return
			}

			// 4. Kiểm tra Redis Blacklist bằng `jti`
			jti, hasJti := claims["jti"].(string)
			if !hasJti || jti == "" {
				http.Error(w, "Unauthorized - Missing JTI claim in token", http.StatusUnauthorized)
				return
			}

			if isBlacklisted(jti) {
				http.Error(w, "Unauthorized - Token has been revoked", http.StatusUnauthorized)
				return
			}

			// 5. Xử lý các Headers - Xóa chống giả mạo danh tính từ Client
			r.Header.Del("X-User-ID")
			r.Header.Del("X-User-Role")

			// Gắn thông tin vừa bóc ra được từ Token chuẩn vào Header cho backend
			// Thông thường ID có thể nằm ở claim "id" hoặc "sub"
			if userID, ok := claims["id"].(string); ok {
				r.Header.Set("X-User-ID", userID)
			} else if subID, ok := claims["sub"].(string); ok {
				r.Header.Set("X-User-ID", subID)
			} else if floatID, ok := claims["id"].(float64); ok {
				r.Header.Set("X-User-ID", fmt.Sprintf("%.0f", floatID))
			}

			if role, ok := claims["role"].(string); ok {
				r.Header.Set("X-User-Role", role)
			}

			// Mọi thứ hoàn hảo, cho phép đi tiếp
			next.ServeHTTP(w, r)
		})
	}
}

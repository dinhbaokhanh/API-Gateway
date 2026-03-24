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

// InitJWT nạp JWT_SECRET từ biến môi trường. Crash nếu không tìm thấy để đảm bảo an toàn.
func InitJWT() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		panic("CRITICAL: Thiếu biến môi trường JWT_SECRET — Gateway từ chối khởi động!")
	}
	jwtSecret = []byte(secret)
}

// AuthMiddlewareProvider tạo middleware xác thực JWT cho từng route cụ thể.
// Kiểm tra: định dạng Bearer, thuật toán HS256, exp/iss/aud/jti, Redis blacklist.
func AuthMiddlewareProvider(jwtCfg config.JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// Lấy và kiểm tra định dạng header Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Unauthorized - Thiếu hoặc sai định dạng Authorization header", http.StatusUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			// Parse token, chỉ chấp nhận HS256 — ngăn chặn tấn công alg:none
			token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("thuật toán ký không hợp lệ: %v", token.Header["alg"])
				}
				return jwtSecret, nil
			},
				jwt.WithValidMethods([]string{"HS256"}),
				jwt.WithExpirationRequired(),
				jwt.WithIssuer(jwtCfg.Issuer),
				jwt.WithAudience(jwtCfg.Audience),
			)

			if err != nil || !token.Valid {
				http.Error(w, "Unauthorized - Token không hợp lệ hoặc đã hết hạn", http.StatusUnauthorized)
				return
			}

			// Đọc claims từ payload
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, "Unauthorized - Không đọc được thông tin token", http.StatusUnauthorized)
				return
			}

			// Kiểm tra jti tồn tại để tra cứu blacklist
			jti, hasJti := claims["jti"].(string)
			if !hasJti || jti == "" {
				http.Error(w, "Unauthorized - Token thiếu claim jti", http.StatusUnauthorized)
				return
			}

			// Từ chối nếu jti đã bị revoke (người dùng đã đăng xuất hoặc token bị cướp)
			if isBlacklisted(jti) {
				http.Error(w, "Unauthorized - Token đã bị thu hồi", http.StatusUnauthorized)
				return
			}

			// Xóa header giả mạo do client tự đặt, sau đó gắn lại từ claims đáng tin cậy
			r.Header.Del("X-User-ID")
			r.Header.Del("X-User-Role")

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

			next.ServeHTTP(w, r)
		})
	}
}

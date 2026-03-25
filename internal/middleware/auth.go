package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/dinhbaokhanh/Final-Project-API-Gateway/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// Removed static jwtSecret var.
// It will be loaded dynamically via os.Getenv("JWT_SECRET")

// InitJWT ensures the environment variable is present on startup
func InitJWT() {
	if os.Getenv("JWT_SECRET") == "" {
		panic("CRITICAL: Thiếu biến môi trường JWT_SECRET — Gateway từ chối khởi động!")
	}
}

// AuthMiddlewareProvider tạo middleware xác thực JWT và kiểm tra RBAC (Role-Based Access Control)
func AuthMiddlewareProvider(jwtCfg config.JWTConfig, requiredRoles []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// Lấy và kiểm tra định dạng header Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Unauthorized - Thiếu hoặc sai định dạng Authorization header", http.StatusUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			// Lấy secret dynamically để không bị cứng
			jwtSecret := []byte(os.Getenv("JWT_SECRET"))

			// Khởi tạo Parser với các cấu hình validation và tùy chọn giữ nguyên số lớn (JSONNumber)
			parser := jwt.NewParser(
				jwt.WithValidMethods([]string{"HS256"}),
				jwt.WithExpirationRequired(),
				jwt.WithIssuer(jwtCfg.Issuer),
				jwt.WithAudience(jwtCfg.Audience),
				jwt.WithJSONNumber(),
			)

			token, err := parser.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("thuật toán ký không hợp lệ: %v", token.Header["alg"])
				}
				return jwtSecret, nil
			})

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

			// Xóa header giả mạo do client tự đặt
			r.Header.Del("X-User-ID")
			r.Header.Del("X-User-Role")

			// Lấy UserID (tránh lỗi trống / float64)
			var userIDStr string
			if userID, ok := claims["id"].(string); ok {
				userIDStr = userID
			} else if subID, ok := claims["sub"].(string); ok {
				userIDStr = subID
			} else if numID, ok := claims["id"].(json.Number); ok {
				userIDStr = numID.String() // Giữ nguyên độ dài, không lo mất số
			}

			// Chặn Bypass nếu ID trống rỗng
			if strings.TrimSpace(userIDStr) == "" {
				http.Error(w, "Unauthorized - Invalid or Empty User ID in Token", http.StatusUnauthorized)
				return
			}
			r.Header.Set("X-User-ID", userIDStr)

			role, hasRole := claims["role"].(string)
			// RBAC Validation (kiểm tra Role)
			if len(requiredRoles) > 0 {
				if !hasRole || role == "" {
					http.Error(w, "Forbidden - Missing Role Claim", http.StatusForbidden)
					return
				}
				
				isAllowed := false
				for _, reqRole := range requiredRoles {
					if role == reqRole {
						isAllowed = true
						break
					}
				}
				if !isAllowed {
					http.Error(w, "Forbidden - Insufficient Permissions", http.StatusForbidden)
					return
				}
			}

			if hasRole && role != "" {
				r.Header.Set("X-User-Role", role)
			}

			next.ServeHTTP(w, r)
		})
	}
}

package middleware

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client

// InitRedis khởi tạo kết nối tới Redis
func InitRedis(addr string) {
	rdb = redis.NewClient(&redis.Options{
		Addr: addr,
	})
	
	// Test kết nối cơ bản, nếu không kết nối được có thể in cảnh báo (trong thực tế có thể log.Fatal)
	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		panic("Không thể kết nối tới Redis: " + err.Error())
	}
}

// RevokeToken đưa một JWT ID (jti) vào danh sách đen cho tới khi nó tự động hết hạn
func RevokeToken(jti string, expireAt time.Time) error {
	if rdb == nil {
		return nil
	}
	
	now := time.Now()
	if expireAt.Before(now) {
		return nil // Đã hết hạn rồi, không cần đưa vào blacklist nữa
	}
	
	ttl := expireAt.Sub(now)
	ctx := context.Background()
	
	// Lưu vào Redis với tiền tố "blacklist:"
	return rdb.Set(ctx, "blacklist:"+jti, "revoked", ttl).Err()
}

// isBlacklisted kiểm tra xem một jti có nằm trong danh sách đen không
func isBlacklisted(jti string) bool {
	if rdb == nil || jti == "" {
		return false
	}
	
	ctx := context.Background()
	val, err := rdb.Get(ctx, "blacklist:"+jti).Result()
	if err == redis.Nil {
		return false // Không tìm thấy trong blacklist
	} else if err != nil {
		return false // Lỗi kết nối redis (để an toàn, có thể chọn return true để block, nhưng ở đây tạm return false)
	}
	
	return val == "revoked"
}

package middleware

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client

// InitRedis khởi tạo kết nối Redis. Crash nếu không kết nối được — blacklist yêu cầu Redis hoạt động.
func InitRedis(addr string) {
	rdb = redis.NewClient(&redis.Options{Addr: addr})

	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		panic("CRITICAL: Không thể kết nối tới Redis tại " + addr + ": " + err.Error())
	}
	log.Printf("[OK] Đã kết nối Redis tại %s\n", addr)
}

// RevokeToken thêm jti vào danh sách đen với TTL bằng thời gian sống còn lại của token.
// Redis tự dọn dẹp sau khi token hết hạn — không tốn bộ nhớ vô thời hạn.
func RevokeToken(jti string, expireAt time.Time) error {
	if rdb == nil {
		return nil
	}
	if expireAt.Before(time.Now()) {
		return nil // Token đã hết hạn, không cần revoke
	}
	return rdb.Set(context.Background(), "blacklist:"+jti, "revoked", time.Until(expireAt)).Err()
}

func isBlacklisted(jti string) bool {
	if rdb == nil || jti == "" {
		return false
	}
	val, err := rdb.Get(context.Background(), "blacklist:"+jti).Result()
	if err == redis.Nil {
		return false
	} else if err != nil {
		log.Printf("[LỖI BẢO MẬT] Redis blacklist lookup failed for jti %s: %v. Báo cáo failed-closed.", jti, err)
		return true // Fail-closed: Coi như token bị reject nếu Redis bị lỗi
	}
	return val == "revoked"
}

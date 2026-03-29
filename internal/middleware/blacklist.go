package middleware

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

// InitRedis khởi tạo kết nối Redis.
func InitRedis(addr string) {
	RedisClient = redis.NewClient(&redis.Options{Addr: addr})

	_, err := RedisClient.Ping(context.Background()).Result()
	if err != nil {
		panic("CRITICAL: Không thể kết nối tới Redis tại " + addr + ": " + err.Error())
	}
	log.Printf("[OK] Đã kết nối Redis tại %s\n", addr)
}

// Redis tự dọn dẹp sau khi token hết hạn.
func RevokeToken(jti string, expireAt time.Time) error {
	if RedisClient == nil {
		return nil
	}
	if expireAt.Before(time.Now()) {
		return nil // Token đã hết hạn, không cần revoke
	}
	return RedisClient.Set(context.Background(), "blacklist:"+jti, "revoked", time.Until(expireAt)).Err()
}

func isBlacklisted(jti string) bool {
	if RedisClient == nil || jti == "" {
		return false
	}
	val, err := RedisClient.Get(context.Background(), "blacklist:"+jti).Result()
	if err == redis.Nil {
		return false
	} else if err != nil {
		log.Printf("[LỖI BẢO MẬT] Redis blacklist lookup failed for jti %s: %v. Báo cáo failed-closed.", jti, err)
		return true
	}
	return val == "revoked"
}

package redis

import (
	"context"
	"encoding/json"
	"log"

	"ludrum/internal/logger"

	"github.com/redis/go-redis/v9"
)

// ==========================
// CLIENT
// ==========================
type RedisClient struct {
	Client *redis.Client
	ctx    context.Context
	prefix string
}

// ==========================
// CONSTRUCTOR
// ==========================
func NewRedisClient(addr string, prefix string) *RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	return &RedisClient{
		Client: rdb,
		ctx:    context.Background(),
		prefix: prefix,
	}
}

// ==========================================
// 🔥 NEW: SNAPSHOT / DELTA PUBLISH (MAIN)
// ==========================================
func (r *RedisClient) PublishPayloadWithType(payload interface{}, payloadType string) {

	data, err := json.Marshal(payload)
	if err != nil {
		log.Println("❌ JSON marshal error (payload):", err)
		return
	}

	channel := r.prefix + ":" + payloadType // snapshot / delta

	// 🔥 ASYNC publish (prevents blocking)
	go func() {
		if err := r.Client.Publish(r.ctx, channel, data).Err(); err != nil {
			logger.Error("redis", "publish payload failed", err, nil)
		}
	}()

	// 🔥 store latest snapshot only
	if payloadType == "snapshot" {
		key := r.prefix + ":latest:payload"

		go func() {
			if err := r.Client.Set(r.ctx, key, data, 0).Err(); err != nil {
				logger.Error("redis", "set latest payload failed", err, nil)
			}
		}()
	}
}

// ==========================
// SNAPSHOT (LEGACY SAFE)
// ==========================
func (r *RedisClient) PublishPayload(payload interface{}) {

	data, err := json.Marshal(payload)
	if err != nil {
		log.Println("❌ JSON marshal error (payload):", err)
		return
	}

	key := r.prefix + ":latest:payload"
	channel := r.prefix + ":payload_stream"

	// store latest
	go func() {
		if err := r.Client.Set(r.ctx, key, data, 0).Err(); err != nil {
			logger.Error("redis", "set latest payload failed", err, nil)
		}
	}()

	// publish
	go func() {
		if err := r.Client.Publish(r.ctx, channel, data).Err(); err != nil {
			logger.Error("redis", "publish payload failed", err, nil)
		}
	}()
}

// ==========================
// STREAM (UNCHANGED)
// ==========================
func (r *RedisClient) PublishPayloadStream(payload interface{}) {

	data, err := json.Marshal(payload)
	if err != nil {
		log.Println("❌ JSON marshal error (stream):", err)
		return
	}

	stream := r.prefix + ":dashboard_stream"

	go func() {
		_, err := r.Client.XAdd(r.ctx, &redis.XAddArgs{
			Stream: stream,
			Values: map[string]interface{}{
				"data": data,
			},
		}).Result()

		if err != nil {
			log.Println("❌ Redis stream error:", err)
			return
		}

		_ = r.Client.XTrimMaxLen(r.ctx, stream, 1000).Err()
	}()
}

// ==========================
// LTP (UNCHANGED)
// ==========================
func (r *RedisClient) PublishLTP(payload interface{}) {

	data, err := json.Marshal(payload)
	if err != nil {
		log.Println("❌ JSON marshal error (ltp):", err)
		return
	}

	key := r.prefix + ":latest:ltp"
	channel := r.prefix + ":ltp_stream"

	go func() {
		if err := r.Client.Set(r.ctx, key, data, 0).Err(); err != nil {
			logger.Error("redis", "set latest ltp failed", err, nil)
		}
	}()

	go func() {
		if err := r.Client.Publish(r.ctx, channel, data).Err(); err != nil {
			logger.Error("redis", "publish ltp failed", err, nil)
		}
	}()
}

// ==========================
// GENERIC (UNCHANGED)
// ==========================
func (r *RedisClient) Publish(channel string, payload interface{}) {

	data, err := json.Marshal(payload)
	if err != nil {
		log.Println("❌ JSON marshal error (generic):", err)
		return
	}

	fullChannel := r.prefix + ":" + channel

	go func() {
		if err := r.Client.Publish(r.ctx, fullChannel, data).Err(); err != nil {
			logger.Error("redis", "generic publish failed", err, nil)
		}
	}()
}

// ==========================
// GET HELPERS
// ==========================
func (r *RedisClient) GetLatestPayload() ([]byte, error) {
	return r.Client.Get(r.ctx, r.prefix+":latest:payload").Bytes()
}

func (r *RedisClient) GetLatestLTP() ([]byte, error) {
	return r.Client.Get(r.ctx, r.prefix+":latest:ltp").Bytes()
}

func (r *RedisClient) GetPrefix() string {
	return r.prefix
}
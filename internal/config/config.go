// Package config provides shared configuration loading for Horizon Core services.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App          AppConfig
	HTTP         HTTPConfig
	GRPC         GRPCConfig
	Database     DatabaseConfig
	Cache        CacheConfig
	MessageQueue MessageQueueConfig
}

type AppConfig struct{ Name, Environment, Version string }
type HTTPConfig struct {
	Address                                string
	ReadTimeout, WriteTimeout, IdleTimeout time.Duration
}
type GRPCConfig struct{ Address string }
type DatabaseConfig struct {
	Driver, DSN                string
	MaxOpenConns, MaxIdleConns int
	ConnMaxLifetime            time.Duration
}
type CacheConfig struct {
	Address, Password string
	DB                int
	DefaultTTL        time.Duration
}
type MessageQueueConfig struct{ URL, Exchange, ClientID string }

func Load() Config {
	return Config{
		App:          AppConfig{Name: get("HORIZON_APP_NAME", "horizon-core"), Environment: get("HORIZON_ENV", "development"), Version: get("HORIZON_VERSION", "dev")},
		HTTP:         HTTPConfig{Address: get("HTTP_ADDRESS", ":8080"), ReadTimeout: duration("HTTP_READ_TIMEOUT", 5*time.Second), WriteTimeout: duration("HTTP_WRITE_TIMEOUT", 10*time.Second), IdleTimeout: duration("HTTP_IDLE_TIMEOUT", 60*time.Second)},
		GRPC:         GRPCConfig{Address: get("GRPC_ADDRESS", ":9090")},
		Database:     DatabaseConfig{Driver: get("DB_DRIVER", "postgres"), DSN: get("DB_DSN", "postgres://horizon:horizon@localhost:5432/horizon?sslmode=disable"), MaxOpenConns: integer("DB_MAX_OPEN_CONNS", 25), MaxIdleConns: integer("DB_MAX_IDLE_CONNS", 5), ConnMaxLifetime: duration("DB_CONN_MAX_LIFETIME", time.Hour)},
		Cache:        CacheConfig{Address: get("CACHE_ADDRESS", "localhost:6379"), Password: get("CACHE_PASSWORD", ""), DB: integer("CACHE_DB", 0), DefaultTTL: duration("CACHE_DEFAULT_TTL", 5*time.Minute)},
		MessageQueue: MessageQueueConfig{URL: get("MQ_URL", "amqp://guest:guest@localhost:5672/"), Exchange: get("MQ_EXCHANGE", "horizon.events"), ClientID: get("MQ_CLIENT_ID", "horizon-core")},
	}
}

func (c Config) Validate() error {
	var missing []string
	if c.App.Name == "" {
		missing = append(missing, "app.name")
	}
	if c.HTTP.Address == "" {
		missing = append(missing, "http.address")
	}
	if c.GRPC.Address == "" {
		missing = append(missing, "grpc.address")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required config: %s", strings.Join(missing, ", "))
	}
	return nil
}

func get(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
func integer(key string, fallback int) int {
	v := get(key, "")
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}
func duration(key string, fallback time.Duration) time.Duration {
	v := get(key, "")
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

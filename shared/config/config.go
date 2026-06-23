package config

import "time"

type Config struct {
	SubscriptionDB SubscriptionDBConfig
	TrackingDB     TrackingDBConfig
	HTTP           HTTPConfig
	GRPC           GRPCConfig
	Redis          RedisConfig
	Email          EmailConfig
	GitHub         GitHubConfig
	Worker         WorkerConfig
	App            AppConfig
}

type SubscriptionDBConfig struct {
	DSN               string        `env:"SUBSCRIPTION_DB_SOURCE"`
	MaxOpenConns      int32         `env:"SUBSCRIPTION_DB_MAX_OPEN_CONNS"`
	MaxConnIdleTime   time.Duration `env:"SUBSCRIPTION_DB_MAX_CONN_IDLE_TIME"`
	HealthCheckPeriod time.Duration `env:"SUBSCRIPTION_DB_HEALTH_CHECK_PERIOD"`
}

type TrackingDBConfig struct {
	DSN               string        `env:"TRACKING_DB_SOURCE"`
	MaxOpenConns      int32         `env:"TRACKING_DB_MAX_OPEN_CONNS"`
	MaxConnIdleTime   time.Duration `env:"TRACKING_DB_MAX_CONN_IDLE_TIME"`
	HealthCheckPeriod time.Duration `env:"TRACKING_DB_HEALTH_CHECK_PERIOD"`
}

type HTTPConfig struct {
	Port            string        `env:"APP_HTTP_PORT"`
	ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT"`
	RequestTimeout  time.Duration `env:"HTTP_REQUEST_TIMEOUT"`
}

type GRPCConfig struct {
	Port string `env:"APP_GRPC_PORT"`
}

type RedisConfig struct {
	Host     string `env:"REDIS_HOST"`
	Port     string `env:"REDIS_PORT"`
	Password string `env:"REDIS_PASSWORD"`
	DB       int    `env:"REDIS_DB"`
}

type EmailConfig struct {
	Host     string `env:"EMAIL_HOST"`
	Port     string `env:"EMAIL_PORT"`
	User     string `env:"EMAIL_USER"`
	Password string `env:"EMAIL_PASSWORD"`
	From     string `env:"EMAIL_FROM"`
}

type GitHubConfig struct {
	Token              string        `env:"GITHUB_TOKEN"`
	HTTPTimeout        time.Duration `env:"GITHUB_HTTP_TIMEOUT"`
	CBMaxRequests      uint32        `env:"GITHUB_CB_MAX_REQUESTS"`
	CBInterval         time.Duration `env:"GITHUB_CB_INTERVAL"`
	CBTimeout          time.Duration `env:"GITHUB_CB_TIMEOUT"`
	CBFailureThreshold uint32        `env:"GITHUB_CB_FAILURE_THRESHOLD"`
}

type WorkerConfig struct {
	NotificationInterval time.Duration `env:"NOTIFICATION_INTERVAL"`
	MaxSendWorkers       int           `env:"WORKER_MAX_SEND_WORKERS"`
	SendTimeout          time.Duration `env:"WORKER_SEND_TIMEOUT"`
	HealthPort           string        `env:"WORKER_HEALTH_PORT"`
}

type AppConfig struct {
	APIKey    string `env:"APP_API_KEY"`
	BaseURL   string `env:"APP_BASE_URL"`
	RateLimit string `env:"RATE_LIMIT"`
}

type RabbitMQConfig struct {
	URL string `env:"RABBITMQ_URL"`
}

type GRPCClientConfig struct {
	SubscriptionAddr string `env:"SUBSCRIPTION_GRPC_ADDR"`
	APIKey           string `env:"APP_API_KEY"`
}

type SenderConfig struct {
	SendTimeout time.Duration `env:"WORKER_SEND_TIMEOUT"`
	HealthPort  string        `env:"SENDER_HEALTH_PORT"`
}

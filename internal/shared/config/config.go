package config

import "time"

type Config struct {
	DB     DBConfig
	HTTP   HTTPConfig
	GRPC   GRPCConfig
	Redis  RedisConfig
	Email  EmailConfig
	GitHub GitHubConfig
	Worker WorkerConfig
	App    AppConfig
}

type DBConfig struct {
	DSN               string        `env:"DB_SOURCE"              envDefault:"postgres://user:password@localhost:5432/userservice?sslmode=disable"`
	MaxOpenConns      int32         `env:"DB_MAX_OPEN_CONNS"      envDefault:"25"`
	MaxConnIdleTime   time.Duration `env:"DB_MAX_CONN_IDLE_TIME"  envDefault:"30m"`
	HealthCheckPeriod time.Duration `env:"DB_HEALTH_CHECK_PERIOD" envDefault:"1m"`
}

type HTTPConfig struct {
	Port            string        `env:"APP_HTTP_PORT"         envDefault:"8080"`
	ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" envDefault:"5s"`
	RequestTimeout  time.Duration `env:"HTTP_REQUEST_TIMEOUT"  envDefault:"5s"`
}

type GRPCConfig struct {
	Port string `env:"APP_GRPC_PORT" envDefault:"9090"`
}

type RedisConfig struct {
	Host     string `env:"REDIS_HOST"     envDefault:"redis"`
	Port     string `env:"REDIS_PORT"     envDefault:"6379"`
	Password string `env:"REDIS_PASSWORD" envDefault:"redis_password"`
	DB       int    `env:"REDIS_DB"       envDefault:"0"`
}

type EmailConfig struct {
	Host     string `env:"EMAIL_HOST"     envDefault:"smtp.example.com"`
	Port     string `env:"EMAIL_PORT"     envDefault:"587"`
	User     string `env:"EMAIL_USER"     envDefault:"smtp_user"`
	Password string `env:"EMAIL_PASSWORD" envDefault:"smtp_password"`
	From     string `env:"EMAIL_FROM"     envDefault:"smtp.example.com"`
}

type GitHubConfig struct {
	Token              string        `env:"GITHUB_TOKEN"                envDefault:""`
	HTTPTimeout        time.Duration `env:"GITHUB_HTTP_TIMEOUT"         envDefault:"10s"`
	CBMaxRequests      uint32        `env:"GITHUB_CB_MAX_REQUESTS"      envDefault:"3"`
	CBInterval         time.Duration `env:"GITHUB_CB_INTERVAL"          envDefault:"5s"`
	CBTimeout          time.Duration `env:"GITHUB_CB_TIMEOUT"           envDefault:"30s"`
	CBFailureThreshold uint32        `env:"GITHUB_CB_FAILURE_THRESHOLD" envDefault:"3"`
}

type WorkerConfig struct {
	NotificationInterval time.Duration `env:"NOTIFICATION_INTERVAL"   envDefault:"1m"`
	MaxSendWorkers       int           `env:"WORKER_MAX_SEND_WORKERS" envDefault:"10"`
	SendTimeout          time.Duration `env:"WORKER_SEND_TIMEOUT"     envDefault:"5s"`
}

type AppConfig struct {
	APIKey    string `env:"APP_API_KEY"  envDefault:"secret"`
	BaseURL   string `env:"APP_BASE_URL" envDefault:"http://localhost:8080"`
	RateLimit string `env:"RATE_LIMIT"   envDefault:"10-S"`
}

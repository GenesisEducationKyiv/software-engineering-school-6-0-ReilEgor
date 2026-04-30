package config

type (
	RedisHostType     string
	RedisPortType     string
	RedisPasswordType string

	HTTPPortType string
	DSNType      string

	EmailHostType     string
	EmailPortType     string
	EmailPasswordType string
	EmailFromType     string
	EmailUserType     string

	ApiKeyType string

	GRPCPortType    string
	AppBaseURLType  string
	GitHubTokenType string
)
type Config struct {
	DSN           DSNType           `env:"DB_SOURCE" envDefault:"postgres://user:password@localhost:5432/userservice?sslmode=disable"`
	HTTPPort      HTTPPortType      `env:"APP_HTTP_PORT" envDefault:"8080"`
	RedisHost     RedisHostType     `env:"REDIS_HOST" envDefault:"redis"`
	RedisPort     RedisPortType     `env:"REDIS_PORT" envDefault:"6379"`
	RedisPassword RedisPasswordType `env:"REDIS_PASSWORD" envDefault:"redis_password"`

	EmailHost     EmailHostType     `env:"EMAIL_HOST" envDefault:"smtp.example.com"`
	EmailPort     EmailPortType     `env:"EMAIL_PORT" envDefault:"587"`
	EmailPassword EmailPasswordType `env:"EMAIL_PASSWORD" envDefault:"smtp_password"`
	EmailFrom     EmailFromType     `env:"EMAIL_FROM" envDefault:"smtp.example.com"`
	EmailUser     EmailUserType     `env:"EMAIL_USER" envDefault:"smtp_user"`

	ApiKey     ApiKeyType     `env:"APP_API_KEY" envDefault:"smtp_api_key"`
	AppBaseURL AppBaseURLType `env:"APP_BASE_URL" envDefault:"https://smtp.example.com"`
	GRPCPort   GRPCPortType   `env:"APP_GRPC_PORT" envDefault:"9090"`

	GitHubToken GitHubTokenType `env:"GITHUB_TOKEN" envDefault:""`
}

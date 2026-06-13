//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/config"
	"github.com/google/wire"
)

func ProvideEmailConfig(cfg Config) config.EmailConfig { return cfg.Email }

type App struct {
}

func InitializeApp(ctx context.Context, cfg Config) (*App, func(), error) {
	wire.Build(
		//ProvideEmailConfig,
		wire.Struct(new(App), "*"),
	)
	return nil, nil, nil
}

package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/tlmanz/allure-hub/internal/app"
	"github.com/tlmanz/allure-hub/pkg/config"
	"github.com/tlmanz/goconf"
)

func main() {
	// Bootstrap logger is used only until config is parsed and the real logger built.
	bootstrap, _ := zap.NewProduction()
	defer bootstrap.Sync() //nolint:errcheck

	if err := goconf.Load(new(config.Config)); err != nil {
		bootstrap.Fatal("config", zap.Error(err))
	}

	log, err := buildLogger(config.Values.Log)
	if err != nil {
		bootstrap.Fatal("logger", zap.Error(err))
	}
	defer log.Sync() //nolint:errcheck

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	a, err := app.New(config.Values, log)
	if err != nil {
		log.Fatal("initialisation failed", zap.Error(err))
	}

	if err := a.Run(ctx); err != nil {
		log.Error("server exited with error", zap.Error(err))
		os.Exit(1)
	}
}

// buildLogger constructs the production zap logger from log config.
func buildLogger(cfg config.LogConfig) (*zap.Logger, error) {
	var lvl zapcore.Level
	if err := lvl.UnmarshalText([]byte(strings.ToLower(cfg.Level))); err != nil {
		return nil, err
	}
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	return zap.Config{
		Level:            zap.NewAtomicLevelAt(lvl),
		Encoding:         strings.ToLower(cfg.Format),
		EncoderConfig:    encCfg,
		OutputPaths:      []string{strings.ToLower(cfg.Output)},
		ErrorOutputPaths: []string{"stderr"},
	}.Build()
}

package app

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/tlmanz/allure-hub/pkg/config"
	notify "github.com/tlmanz/go-notify"
	"github.com/tlmanz/go-notify/memory"
	redisnotify "github.com/tlmanz/go-notify/redis"
)

func newNotifier(cfg config.NotificationConfig, log *zap.Logger) (*notify.Notifier, func(), error) {
	var repo notify.Repository
	closeRepo := func() {}

	if cfg.RedisURL != "" {
		redisRepo, err := redisnotify.NewFromURL(context.Background(), cfg.RedisURL, redisnotify.Options{
			KeyPrefix: cfg.RedisKeyPrefix,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("open notification redis repository: %w", err)
		}
		repo = redisRepo
		closeRepo = func() {
			if err := redisRepo.Close(); err != nil {
				log.Warn("close notification redis repository failed", zap.Error(err))
			}
		}
		log.Info("notifications repository configured", zap.String("backend", "redis"), zap.String("keyPrefix", cfg.RedisKeyPrefix))
	} else {
		repo = memory.New(memory.Options{MaxItems: 1000})
		log.Info("notifications repository configured", zap.String("backend", "memory"))
	}

	retentionDays := cfg.RetentionDays
	if retentionDays < 0 {
		log.Warn("negative notification retention configured, defaulting to 0", zap.Int("retentionDays", retentionDays))
		retentionDays = 0
	}
	notifier, err := notify.New(repo, notify.Options{
		RetentionDays:     retentionDays,
		EnabledTransports: []notify.Transport{notify.TransportSSE},
		LogHandler:        newNotifyLogHandler(log),
	})
	if err != nil {
		closeRepo()
		return nil, nil, fmt.Errorf("create notifier: %w", err)
	}
	return notifier, closeRepo, nil
}

func newNotifyLogHandler(log *zap.Logger) func(notify.LogEvent) {
	notifyLog := log.With(zap.String("component", "go-notify"))
	return func(e notify.LogEvent) {
		fields := make([]zap.Field, 0, len(e.Fields)+2)
		fields = append(fields, zap.Time("eventTime", e.Time))
		if e.Err != nil {
			fields = append(fields, zap.Error(e.Err))
		}
		for k, v := range e.Fields {
			fields = append(fields, zap.Any(k, v))
		}

		switch e.Level {
		case notify.LogLevelDebug:
			notifyLog.Debug(e.Message, fields...)
		case notify.LogLevelInfo:
			notifyLog.Info(e.Message, fields...)
		case notify.LogLevelWarn:
			notifyLog.Warn(e.Message, fields...)
		case notify.LogLevelError:
			notifyLog.Error(e.Message, fields...)
		default:
			notifyLog.Info(e.Message, fields...)
		}
	}
}

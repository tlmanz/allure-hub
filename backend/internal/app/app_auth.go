package app

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/tlmanz/allure-hub/internal/repository"
	"github.com/tlmanz/allure-hub/internal/transport"
	"github.com/tlmanz/allure-hub/pkg/config"
	"github.com/tlmanz/authkit"
)

func newAuth(cfg config.AuthConfig, db *repository.DB, keyStore authkit.APIKeyValidator, log *zap.Logger) (*authkit.Auth, *authkit.LayeredPolicyProvider, int, error) {
	roleStore := repository.NewRoleStore(db)
	zapLogger := transport.NewZapAuthLogger(log)

	provider, err := authkit.NewLayeredProvider(cfg.PolicyFile, roleStore,
		authkit.WithLogger(zapLogger),
	)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("authkit: layered provider: %w", err)
	}

	var providers []authkit.ProviderConfig
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		providers = append(providers, authkit.ProviderConfig{
			Name:         "google",
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
		})
	}

	auth, err := authkit.New(authkit.Config{
		Providers:       providers,
		CallbackBaseURL: cfg.BaseURL,
		SessionSecret:   cfg.SessionSecret,
		SecureCookie:    cfg.SecureCookie,
		AfterLoginURL:   cfg.AfterLoginURL,
		AfterLogoutURL:  cfg.AfterLogoutURL,
		RBAC:            authkit.RBACConfig{Provider: provider},
		Logger:          zapLogger,
		APIKeyValidator: keyStore,
	})
	if err != nil {
		return nil, nil, 0, fmt.Errorf("authkit: %w", err)
	}
	return auth, provider, len(providers), nil
}

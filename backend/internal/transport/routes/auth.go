package routes

import (
	"context"
	"time"

	"github.com/tlmanz/allure-hub/internal/domain"
	kit "github.com/tlmanz/authkit"
)

// upsertTrackedUser persists an OAuth user's login record asynchronously.
func upsertTrackedUser(ctx context.Context, repo domain.TrackedUserRepository, u *kit.User) {
	if repo == nil || u == nil {
		return
	}
	now := time.Now().UTC()
	_ = repo.Upsert(ctx, &domain.TrackedUser{
		Email:        u.Email,
		Name:         u.Name,
		AvatarURL:    u.AvatarURL,
		Provider:     u.Provider,
		Role:         u.Role,
		FirstLoginAt: now,
		LastLoginAt:  now,
	})
}

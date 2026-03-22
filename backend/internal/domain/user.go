package domain

import (
	"errors"
	"time"
)

var ErrTrackedUserNotFound = errors.New("tracked user not found")

// TrackedUser is an OAuth user who has logged into allure-hub.
// Created on first login; last_login_at updated on every subsequent login.
type TrackedUser struct {
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	AvatarURL    string    `json:"avatarUrl"`
	Provider     string    `json:"provider"`
	Role         string    `json:"role"`
	FirstLoginAt time.Time `json:"firstLoginAt"`
	LastLoginAt  time.Time `json:"lastLoginAt"`
}

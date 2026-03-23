package domain

import "context"

// EnvironmentRepository is the persistence contract for Environment aggregates.
type EnvironmentRepository interface {
	Create(ctx context.Context, e *Environment) error
	Get(ctx context.Context, id string) (*Environment, error)
	List(ctx context.Context) ([]*Environment, error)
	// CountProjectsBatch returns project counts keyed by env ID in one query (M-09).
	CountProjectsBatch(ctx context.Context, envIDs []string) (map[string]int, error)
	Update(ctx context.Context, id, name, icon string) error
	Delete(ctx context.Context, id string) error
}

// ProjectRepository is the persistence contract for Project aggregates.
// Implementations live in the infrastructure layer.
type ProjectRepository interface {
	Create(ctx context.Context, p *Project) error
	Get(ctx context.Context, envID, id string) (*Project, error)
	List(ctx context.Context, envID string) ([]*Project, error)
	Delete(ctx context.Context, envID, id string) error
}

// BuildRepository is the persistence contract for Build aggregates.
type BuildRepository interface {
	Save(ctx context.Context, b *Build) error
	GetByBuildID(ctx context.Context, envID, projectID, buildID string) (*Build, error)
	// BatchStatsByProject returns count + latest build per project in two queries (M-08).
	BatchStatsByProject(ctx context.Context, envID string, projectIDs []string) (map[string]*ProjectBatchStats, error)
	ListByProject(ctx context.Context, envID, projectID string) ([]*Build, error)
	ListByProjectPaged(ctx context.Context, envID, projectID, filter string, limit, offset int) ([]*Build, error)
	CountByProjectFiltered(ctx context.Context, envID, projectID, filter string) (int, error)
	CountByProject(ctx context.Context, envID, projectID string) (int, error)
	LatestByProject(ctx context.Context, envID, projectID string) (*Build, error)
	StatsForProject(ctx context.Context, envID, projectID string) (*BuildStats, error)
	Delete(ctx context.Context, envID, projectID, buildID string) error
	DeleteByProject(ctx context.Context, envID, projectID string) error
}

// UploadSessionRepository is the persistence contract for UploadSession aggregates.
type UploadSessionRepository interface {
	Create(ctx context.Context, s *UploadSession) error
	Update(ctx context.Context, s *UploadSession) error
	// IncrementReceivedChunks atomically increments received_chunks by 1 and
	// returns the updated session. Used by SaveChunk to avoid a racy
	// read-modify-write (M-05).
	IncrementReceivedChunks(ctx context.Context, uploadID string) (*UploadSession, error)
	GetByUploadID(ctx context.Context, uploadID string) (*UploadSession, error)
	GetByBuild(ctx context.Context, projectID, buildID string) (*UploadSession, error)
	ListRecent(ctx context.Context, limit int) ([]*UploadSession, error)
	GetByID(ctx context.Context, id string) (*UploadSession, error)
	Delete(ctx context.Context, id string) error
	DeleteByProject(ctx context.Context, projectID string) error
	DeleteByEnv(ctx context.Context, envID string) error
}

// TrackedUserRepository tracks OAuth users who have logged into allure-hub.
type TrackedUserRepository interface {
	Upsert(ctx context.Context, u *TrackedUser) error
	List(ctx context.Context) ([]*TrackedUser, error)
	// Search returns users whose email or name match query (case-insensitive
	// substring). An empty query matches all users.
	Search(ctx context.Context, query string, limit, offset int) ([]*TrackedUser, error)
	CountSearch(ctx context.Context, query string) (int, error)
}

// APIKeyRepository is the persistence contract for API keys.
type APIKeyRepository interface {
	Create(ctx context.Context, k *APIKey) error
	GetByHash(ctx context.Context, keyHash string) (*APIKey, error)
	List(ctx context.Context) ([]*APIKey, error)
	// Search returns keys whose name or creator match query (case-insensitive
	// substring). An empty query matches all keys.
	Search(ctx context.Context, query string, limit, offset int) ([]*APIKey, error)
	CountSearch(ctx context.Context, query string) (int, error)
	UpdateLastUsed(ctx context.Context, id string) error
	Revoke(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
}

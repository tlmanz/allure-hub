package repository

import (
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func migrate(db *DB, log *zap.Logger) error {
	goose.SetBaseFS(migrationsFS)
	goose.SetLogger(&gooseLogger{log: log})

	dialect := "sqlite3"
	if db.driver == "postgres" {
		dialect = "postgres"
	}

	if err := goose.SetDialect(dialect); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.Up(db.DB, "migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

// gooseLogger adapts *zap.Logger to the goose.Logger interface.
type gooseLogger struct {
	log *zap.Logger
}

func (l *gooseLogger) Printf(format string, v ...interface{}) {
	l.log.Sugar().Infof(format, v...)
}

func (l *gooseLogger) Fatalf(format string, v ...interface{}) {
	l.log.Sugar().Errorf(format, v...)
}

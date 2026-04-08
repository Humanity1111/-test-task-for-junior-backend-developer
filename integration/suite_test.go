//go:build integration

// Интеграционные тесты: реальная PostgreSQL (Docker/Testcontainers) + httptest.Server.
// Запуск: go test -tags=integration -count=1 ./integration/...
// Нужен Docker. После клонирования: go mod tidy.

package integration

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	infrastructurepostgres "example.com/taskservice/internal/infrastructure/postgres"
	postgresrepo "example.com/taskservice/internal/repository/postgres"
	transporthttp "example.com/taskservice/internal/transport/http"
	swaggerdocs "example.com/taskservice/internal/transport/http/docs"
	httphandlers "example.com/taskservice/internal/transport/http/handlers"
	"example.com/taskservice/internal/usecase/task"
)

var (
	integrationServerURL string
	integrationPool      *pgxpool.Pool
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Fprintln(os.Stderr, "integration TestMain: runtime.Caller failed")
		os.Exit(1)
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), ".."))

	migrations := []string{
		filepath.Join(repoRoot, "migrations", "0001_create_tasks.up.sql"),
		filepath.Join(repoRoot, "migrations", "0002_add_recurrence.up.sql"),
	}

	pgC, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.BasicWaitStrategies(),
		postgres.WithDatabase("taskservice"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.WithInitScripts(migrations...),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start postgres container: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		termCtx := context.Background()
		if err := pgC.Terminate(termCtx); err != nil {
			fmt.Fprintf(os.Stderr, "terminate postgres: %v\n", err)
		}
	}()

	dsn, err := pgC.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "postgres connection string: %v\n", err)
		os.Exit(1)
	}

	pool, err := infrastructurepostgres.Open(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open db pool: %v\n", err)
		os.Exit(1)
	}

	repo := postgresrepo.New(pool)
	svc := task.NewService(repo)
	h := httphandlers.NewTaskHandler(svc)
	docs := swaggerdocs.NewHandler()
	r := transporthttp.NewRouter(h, docs)
	srv := httptest.NewServer(r)
	integrationServerURL = srv.URL
	integrationPool = pool

	code := m.Run()

	srv.Close()
	pool.Close()

	os.Exit(code)
}

func truncateTasks(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	_, err := integrationPool.Exec(ctx, `TRUNCATE tasks RESTART IDENTITY`)
	if err != nil {
		t.Fatalf("truncate tasks: %v", err)
	}
}

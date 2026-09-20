package repository_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/amrrasi/fits/internal/models"
	"github.com/amrrasi/fits/internal/repository"
)

// Integration tests need a migrated PostgreSQL:
//
//	TEST_DATABASE_URL=postgres://user:pass@localhost:5432/fits_db?sslmode=disable go test ./internal/repository/
func setup(t *testing.T) (*repository.UserRepository, *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return repository.NewUserRepository(pool), pool
}

func mkUser(t *testing.T, r *repository.UserRepository, role models.Role) int64 {
	t.Helper()
	email := fmt.Sprintf("it-%d-%s@test.local", time.Now().UnixNano(), role)
	id, err := r.CreateUser(context.Background(), email, "$2a$12$notarealhashnotarealhashnotarealhashnotarealhashnotare", "IT User", role)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.DeleteUserGuardedForTest(context.Background(), id) })
	return id
}

// Two admins demoted at the same time must never leave the system without an admin.
func TestLastAdminGuardUnderConcurrency(t *testing.T) {
	r, pool := setup(t)
	ctx := context.Background()
	// isolate: make every pre-existing active admin inactive inside this test's view is not possible,
	// so we count how many admins exist and demote ALL of them concurrently.
	var ids []int64
	rows, _ := pool.Query(ctx, `SELECT id FROM users WHERE role='admin' AND is_active`)
	for rows.Next() {
		var id int64
		_ = rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()
	a, b := mkUser(t, r, models.RoleAdmin), mkUser(t, r, models.RoleAdmin)
	ids = append(ids, a, b)

	var wg sync.WaitGroup
	var mu sync.Mutex
	okCount, lastAdminErrs := 0, 0
	for _, id := range ids {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			err := r.UpdateUserGuarded(ctx, id, "IT User", models.RoleViewer, true)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				okCount++
			case errors.Is(err, repository.ErrLastAdmin):
				lastAdminErrs++
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}(id)
	}
	wg.Wait()
	n, err := r.CountActiveAdmins(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("no active admin left (ok=%d, blocked=%d)", okCount, lastAdminErrs)
	}
	if lastAdminErrs < 1 {
		t.Fatalf("expected the guard to block at least one demotion (ok=%d)", okCount)
	}
	// restore the survivors' role so the DB is left as we found it
	for _, id := range ids {
		_ = r.UpdateUserGuarded(ctx, id, "IT User", models.RoleAdmin, true)
	}
}

// A refresh token must be single-use even when presented concurrently.
func TestConsumeSessionIsSingleUse(t *testing.T) {
	r, _ := setup(t)
	ctx := context.Background()
	uid := mkUser(t, r, models.RoleViewer)
	hash := fmt.Sprintf("hash-%d", time.Now().UnixNano())
	if _, err := r.CreateSession(ctx, uid, hash, "ua", "127.0.0.1", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	won := 0
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := r.ConsumeSession(ctx, hash); err == nil {
				mu.Lock()
				won++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if won != 1 {
		t.Fatalf("token consumed %d times, want exactly 1", won)
	}
}

func TestSessionsAreOwnedByTheirUser(t *testing.T) {
	r, _ := setup(t)
	ctx := context.Background()
	owner, other := mkUser(t, r, models.RoleViewer), mkUser(t, r, models.RoleViewer)
	if _, err := r.CreateSession(ctx, owner, fmt.Sprintf("h-%d", time.Now().UnixNano()), "ua", "1.1.1.1", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	list, err := r.ListSessions(ctx, owner)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %v", list, err)
	}
	if ok, _ := r.DeleteSessionByID(ctx, other, list[0].ID); ok {
		t.Fatal("a user must not be able to revoke someone else's session")
	}
	if ok, _ := r.DeleteSessionByID(ctx, owner, list[0].ID); !ok {
		t.Fatal("owner must be able to revoke their own session")
	}
}

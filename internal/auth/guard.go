package auth

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/amrrasi/fits/internal/models"
	"github.com/amrrasi/fits/internal/repository"
)

var ErrAccountInactive = errors.New("حساب کاربری غیرفعال یا حذف شده است")

type State struct {
	User       *models.User
	Perms      []string
	MustChange bool
}

type guardEntry struct {
	st  *State
	exp time.Time
}

// Guard re-checks (with a very short cache) that the token's user still exists, is active,
// and returns the CURRENT role/permissions - so revocations take effect immediately.
type Guard struct {
	users *repository.UserRepository
	rbac  *repository.RBACRepository
	ttl   time.Duration
	mu    sync.Mutex
	cache map[int64]guardEntry
}

func NewGuard(u *repository.UserRepository, r *repository.RBACRepository) *Guard {
	return &Guard{users: u, rbac: r, ttl: 10 * time.Second, cache: map[int64]guardEntry{}}
}

func (g *Guard) Invalidate(id int64) { g.mu.Lock(); delete(g.cache, id); g.mu.Unlock() }
func (g *Guard) InvalidateAll()      { g.mu.Lock(); g.cache = map[int64]guardEntry{}; g.mu.Unlock() }

func (g *Guard) Check(ctx context.Context, id int64) (*State, error) {
	g.mu.Lock()
	if e, ok := g.cache[id]; ok && time.Now().Before(e.exp) {
		g.mu.Unlock()
		return e.st, nil
	}
	g.mu.Unlock()

	u, err := g.users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrAccountInactive
		}
		return nil, err
	}
	if !u.IsActive {
		return nil, ErrAccountInactive
	}
	perms, err := g.rbac.GetUserPermissionCodes(ctx, id)
	if err != nil {
		return nil, err
	}
	must, err := g.users.GetMustChange(ctx, id)
	if err != nil {
		return nil, err
	}
	st := &State{User: u, Perms: perms, MustChange: must}
	g.mu.Lock()
	if len(g.cache) > 5000 {
		g.cache = map[int64]guardEntry{}
	}
	g.cache[id] = guardEntry{st: st, exp: time.Now().Add(g.ttl)}
	g.mu.Unlock()
	return st, nil
}

package audit

import (
	"context"
	"net/http"
	"time"

	"github.com/amrrasi/fits/internal/logger"
	"github.com/amrrasi/fits/internal/middleware"
	"github.com/amrrasi/fits/internal/repository"
)

type Recorder struct{ repo *repository.UserRepository }

func New(repo *repository.UserRepository) *Recorder { return &Recorder{repo: repo} }

func (r *Recorder) Log(req *http.Request, userID *int64, action, entityType, entityID string, oldV, newV interface{}) {
	if r == nil || r.repo == nil {
		return
	}
	ip := middleware.ClientIP(req)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := r.repo.InsertAuditLog(ctx, userID, action, entityType, entityID, oldV, newV, ip, middleware.GetRequestID(req.Context())); err != nil {
		logger.S().Warnw("audit log write failed", "action", action, "err", err)
	}
}

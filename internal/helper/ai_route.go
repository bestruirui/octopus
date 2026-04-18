package helper

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
)

const aiRouteTaskTimeout = 30 * time.Minute

var aiRouteProgress sync.Map

func StartGenerateAIRoute(req model.GenerateAIRouteRequest) (*model.GenerateAIRouteProgress, error) {
	id := uuid.NewString()
	progress := &model.GenerateAIRouteProgress{
		ID:      id,
		Scope:   req.Scope,
		GroupID: req.GroupID,
	}
	storeAIRouteProgress(progress)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), aiRouteTaskTimeout)
		defer cancel()

		next := cloneAIRouteProgress(progress)
		result, err := op.GenerateAIRoute(ctx, req)
		next.Done = true
		if err != nil {
			next.Message = err.Error()
			storeAIRouteProgress(&next)
			return
		}

		next.Result = cloneAIRouteResult(result)
		if result != nil {
			next.Scope = result.Scope
			next.GroupID = result.GroupID
		}
		storeAIRouteProgress(&next)
	}()

	cloned := cloneAIRouteProgress(progress)
	return &cloned, nil
}

func GetGenerateAIRouteProgress(id string) (*model.GenerateAIRouteProgress, bool) {
	if id == "" {
		return nil, false
	}

	value, ok := aiRouteProgress.Load(id)
	if !ok {
		return nil, false
	}

	progress, ok := value.(model.GenerateAIRouteProgress)
	if !ok {
		return nil, false
	}

	cloned := cloneAIRouteProgress(&progress)
	return &cloned, true
}

func storeAIRouteProgress(progress *model.GenerateAIRouteProgress) {
	if progress == nil || progress.ID == "" {
		return
	}
	aiRouteProgress.Store(progress.ID, cloneAIRouteProgress(progress))
}

func cloneAIRouteProgress(progress *model.GenerateAIRouteProgress) model.GenerateAIRouteProgress {
	if progress == nil {
		return model.GenerateAIRouteProgress{}
	}

	cloned := *progress
	cloned.Result = cloneAIRouteResult(progress.Result)
	return cloned
}

func cloneAIRouteResult(result *model.GenerateAIRouteResult) *model.GenerateAIRouteResult {
	if result == nil {
		return nil
	}

	cloned := *result
	return &cloned
}

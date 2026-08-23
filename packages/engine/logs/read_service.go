package logs

import (
	"context"
	"errors"
	"fmt"
)

// Reader provides independent, read-only access to persisted structured logs.
type Reader interface {
	List(context.Context, ListLogsRequest) (LogPage, error)
	Overview(context.Context) (LogOverview, error)
}

// ReadService applies shared query rules before accessing persistent storage.
type ReadService struct {
	repository Reader
}

// NewReadService creates the reusable structured-log read API.
func NewReadService(repository Reader) *ReadService {
	return &ReadService{repository: repository}
}

// List returns one normalized, deterministic page of logs.
func (s *ReadService) List(ctx context.Context, request ListLogsRequest) (LogPage, error) {
	if err := s.ready(ctx); err != nil {
		return LogPage{}, err
	}

	normalized, err := NormalizeListLogsRequest(request)
	if err != nil {
		return LogPage{}, err
	}
	page, err := s.repository.List(ctx, normalized)
	if err != nil {
		return LogPage{}, fmt.Errorf("list logs: %w", err)
	}
	return page, nil
}

// Overview returns aggregate metadata for all persisted logs.
func (s *ReadService) Overview(ctx context.Context) (LogOverview, error) {
	if err := s.ready(ctx); err != nil {
		return LogOverview{}, err
	}

	overview, err := s.repository.Overview(ctx)
	if err != nil {
		return LogOverview{}, fmt.Errorf("read log overview: %w", err)
	}
	return overview, nil
}

func (s *ReadService) ready(ctx context.Context) error {
	if ctx == nil {
		return errors.New("log read context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.repository == nil {
		return errors.New("log read repository is required")
	}
	return nil
}

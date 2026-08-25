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

// ReadRepository supplies raw log queries and highlighted-column settings.
type ReadRepository interface {
	Reader
	HighlightRepository
}

// ReadService applies shared query rules before accessing persistent storage.
type ReadService struct {
	repository ReadRepository
}

// NewReadService creates the reusable structured-log read API.
func NewReadService(repository ReadRepository) *ReadService {
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
	settings, err := s.repository.HighlightSettings(ctx, normalized.Filter.Applications)
	if err != nil {
		return LogPage{}, fmt.Errorf("read log highlights: %w", err)
	}
	page, err = applyHighlightSettings(page, settings)
	if err != nil {
		return LogPage{}, fmt.Errorf("apply log highlights: %w", err)
	}
	return page, nil
}

// HighlightConfiguration returns every stored application together with its
// selectable fields and current highlighted field.
func (s *ReadService) HighlightConfiguration(ctx context.Context) ([]ApplicationHighlight, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	configuration, err := s.repository.HighlightConfiguration(ctx)
	if err != nil {
		return nil, fmt.Errorf("read log highlight configuration: %w", err)
	}
	return configuration, nil
}

// SaveHighlightSettings validates and atomically persists application settings.
func (s *ReadService) SaveHighlightSettings(ctx context.Context, settings []HighlightSetting) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	configuration, err := s.repository.HighlightConfiguration(ctx)
	if err != nil {
		return fmt.Errorf("read log highlight configuration: %w", err)
	}
	normalized, err := normalizeHighlightSettings(configuration, settings)
	if err != nil {
		return err
	}
	if err := s.repository.SaveHighlightSettings(ctx, normalized); err != nil {
		return fmt.Errorf("save log highlight settings: %w", err)
	}
	return nil
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

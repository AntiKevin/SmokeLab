// Nome: app.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Define a estrutura principal da aplicação visual em Wails, concentrando
// o estado e os métodos expostos ao frontend para que a GUI possa interagir com
// funcionalidades do aplicativo sem mover regras de negócio para a apresentação.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"SmokeLab/packages/engine/logs"
	"SmokeLab/packages/engine/storage"
	"SmokeLab/packages/engine/storage/localdb"
)

// App struct
type App struct {
	mu               sync.RWMutex
	ctx              context.Context
	db               *sql.DB
	logReadService   *logs.ReadService
	startupErr       error
	activeListID     uint64
	activeListCancel context.CancelFunc
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.ctx = ctx
	database, err := localdb.Open(ctx, localdb.DefaultPath())
	if err != nil {
		a.startupErr = fmt.Errorf("não foi possível abrir o banco local de logs: %w", err)
		return
	}

	repository, err := storage.NewLogReadRepository(ctx, database)
	if err != nil {
		_ = database.Close()
		a.startupErr = fmt.Errorf("não foi possível preparar a leitura dos logs: %w", err)
		return
	}

	a.db = database
	a.logReadService = logs.NewReadService(repository)
	a.startupErr = nil
}

// shutdown releases the shared local database connection.
func (a *App) shutdown(context.Context) {
	a.mu.Lock()
	if a.activeListCancel != nil {
		a.activeListCancel()
	}
	database := a.db

	a.db = nil
	a.logReadService = nil
	a.ctx = nil
	a.activeListCancel = nil
	a.activeListID++
	a.mu.Unlock()

	if database != nil {
		_ = database.Close()
	}
}

// ListLogs adapts one frontend request to the reusable engine read service.
func (a *App) ListLogs(request logs.ListLogsRequest) (logs.LogPage, error) {
	a.mu.Lock()
	if a.startupErr != nil {
		err := a.startupErr
		a.mu.Unlock()
		return logs.LogPage{}, err
	}
	if a.ctx == nil || a.logReadService == nil {
		a.mu.Unlock()
		return logs.LogPage{}, errors.New("o serviço de leitura de logs não está disponível")
	}

	if a.activeListCancel != nil {
		a.activeListCancel()
	}
	queryContext, cancel := context.WithCancel(a.ctx)
	a.activeListID++
	requestID := a.activeListID
	a.activeListCancel = cancel
	service := a.logReadService
	a.mu.Unlock()

	page, err := service.List(queryContext, request)
	cancel()

	a.mu.Lock()
	if a.activeListID == requestID {
		a.activeListCancel = nil
	}
	a.mu.Unlock()

	if err != nil {
		return logs.LogPage{}, fmt.Errorf("não foi possível listar os logs: %w", err)
	}
	return page, nil
}

// GetLogOverview returns aggregate log metadata to the frontend.
func (a *App) GetLogOverview() (logs.LogOverview, error) {
	a.mu.RLock()
	if a.startupErr != nil {
		err := a.startupErr
		a.mu.RUnlock()
		return logs.LogOverview{}, err
	}
	if a.ctx == nil || a.logReadService == nil {
		a.mu.RUnlock()
		return logs.LogOverview{}, errors.New("o serviço de leitura de logs não está disponível")
	}
	ctx := a.ctx
	service := a.logReadService
	a.mu.RUnlock()

	overview, err := service.Overview(ctx)
	if err != nil {
		return logs.LogOverview{}, fmt.Errorf("não foi possível carregar o resumo dos logs: %w", err)
	}
	return overview, nil
}

// GetLogHighlightConfiguration returns detected fields and saved selections for
// the configuration panel.
func (a *App) GetLogHighlightConfiguration() ([]logs.ApplicationHighlight, error) {
	a.mu.RLock()
	if a.startupErr != nil {
		err := a.startupErr
		a.mu.RUnlock()
		return nil, err
	}
	if a.ctx == nil || a.logReadService == nil {
		a.mu.RUnlock()
		return nil, errors.New("o serviço de leitura de logs não está disponível")
	}
	ctx := a.ctx
	service := a.logReadService
	a.mu.RUnlock()

	configuration, err := service.HighlightConfiguration(ctx)
	if err != nil {
		return nil, fmt.Errorf("não foi possível carregar as colunas destacadas: %w", err)
	}
	return configuration, nil
}

// SaveLogHighlightSettings persists the highlighted field selected for each
// application.
func (a *App) SaveLogHighlightSettings(settings []logs.HighlightSetting) error {
	a.mu.RLock()
	if a.startupErr != nil {
		err := a.startupErr
		a.mu.RUnlock()
		return err
	}
	if a.ctx == nil || a.logReadService == nil {
		a.mu.RUnlock()
		return errors.New("o serviço de leitura de logs não está disponível")
	}
	ctx := a.ctx
	service := a.logReadService
	a.mu.RUnlock()

	if err := service.SaveHighlightSettings(ctx, settings); err != nil {
		return fmt.Errorf("não foi possível salvar as colunas destacadas: %w", err)
	}
	return nil
}

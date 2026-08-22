// Nome: app.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Define a estrutura principal da aplicação visual em Wails, concentrando
// o estado e os métodos expostos ao frontend para que a GUI possa interagir com
// funcionalidades do aplicativo sem mover regras de negócio para a apresentação.
package main

import (
	"context"

	"SmokeLab/packages/engine"
)

// App struct
type App struct {
	ctx             context.Context
	greetingService *engine.GreetingService
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		greetingService: engine.NewGreetingService(),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return a.greetingService.Greet(name)
}

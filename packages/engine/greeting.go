// Nome: greeting.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Mantém uma funcionalidade simples de saudação no engine, servindo como
// comportamento reutilizável exposto às interfaces sem acoplar a regra a detalhes
// da GUI, CLI ou infraestrutura externa.
package engine

import "fmt"

type GreetingService struct{}

func NewGreetingService() *GreetingService {
	return &GreetingService{}
}

func (s *GreetingService) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

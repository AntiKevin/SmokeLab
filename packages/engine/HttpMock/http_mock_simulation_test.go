// Nome: http_mock_simulation_test.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Valida a simulação de mocks HTTP, cobrindo correspondência de rotas,
// seleção de payloads, respostas esperadas e falhas de configuração para garantir
// que o comportamento simulado continue consistente.
package HttpMock

import "testing"

func TestHttpMockModelSimulateSelectsResponseByPayload(t *testing.T) {
	t.Parallel()

	mock, err := NewHttpMockModel(
		"http://localhost/payload",
		8080,
		Response{ID: "success", Body: []byte("ok")},
		Response{ID: "failure", StatusCode: 400, Body: []byte("bad")},
	)
	if err != nil {
		t.Fatalf("expected mock to be valid: %v", err)
	}

	if err := mock.AddPayload(Payload{ResponseID: "success", Body: []byte("known")}); err != nil {
		t.Fatalf("expected payload to be valid: %v", err)
	}
	if err := mock.AddPayload(Payload{ResponseID: "failure", Body: []byte("invalid")}); err != nil {
		t.Fatalf("expected payload to be valid: %v", err)
	}

	response, err := mock.Simulate(MockRequest{Body: []byte("invalid")})
	if err != nil {
		t.Fatalf("expected internal request to be simulated: %v", err)
	}

	if response.ID != "failure" {
		t.Fatalf("expected failure response, got %q", response.ID)
	}

	if response.EffectiveStatusCode() != 400 {
		t.Fatalf("expected status 400, got %d", response.EffectiveStatusCode())
	}
}

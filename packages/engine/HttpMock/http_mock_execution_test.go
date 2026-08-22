// Nome: http_mock_execution_test.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Testa os cenários de execução dos mocks HTTP, verificando preparação,
// respostas produzidas e tratamento de configurações inválidas para evitar regressões
// no fluxo operacional do domínio.
package HttpMock

import "testing"

func TestHttpMockModelExecuteInternalSimulatesWithoutHTTP(t *testing.T) {
	t.Parallel()

	mock := HttpMockModel{
		URL:               "http://localhost/internal",
		Port:              8080,
		Method:            "post",
		CommunicationType: Internal,
		Headers:           map[string]string{"X-Token": "secret"},
		Responses: []Response{
			{
				ID:         "success",
				StatusCode: 202,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       []byte(`{"internal":true}`),
			},
		},
	}

	execution, err := mock.Execute()
	if err != nil {
		t.Fatalf("expected internal mock to execute: %v", err)
	}

	select {
	case err := <-execution.Done():
		if err != nil {
			t.Fatalf("expected internal execution to finish cleanly: %v", err)
		}
	default:
		t.Fatal("expected internal execution to avoid running an HTTP server")
	}

	response, err := execution.Simulate(MockRequest{
		Method:  "POST",
		Headers: map[string]string{"x-token": "secret"},
		Body:    []byte(`{"request":true}`),
	})
	if err != nil {
		t.Fatalf("expected internal request to be simulated: %v", err)
	}

	if response.EffectiveStatusCode() != 202 {
		t.Fatalf("expected status 202, got %d", response.EffectiveStatusCode())
	}

	if response.Headers["Content-Type"] != "application/json" {
		t.Fatalf("expected JSON response header, got %q", response.Headers["Content-Type"])
	}

	if string(response.Body) != `{"internal":true}` {
		t.Fatalf("expected internal response body, got %q", string(response.Body))
	}
}

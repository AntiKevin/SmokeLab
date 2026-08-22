// Nome: http_mock_model_test.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Exercita o modelo de domínio dos mocks HTTP, garantindo que estruturas,
// valores padrão e relações entre entidades representem corretamente os cenários
// usados pelas etapas de validação, simulação e execução.
package HttpMock

import "testing"

func TestNewHttpMockModelAcceptsValidConfiguration(t *testing.T) {
	t.Parallel()

	mock, err := NewHttpMockModel("http://localhost", 8080, Response{
		ID:         "success",
		StatusCode: 201,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       []byte(`{"ok":true}`),
	})
	if err != nil {
		t.Fatalf("expected valid mock: %v", err)
	}

	response, found := mock.ResponseByID("success")
	if !found {
		t.Fatal("expected response to be registered")
	}

	if response.EffectiveStatusCode() != 201 {
		t.Fatalf("expected status 201, got %d", response.EffectiveStatusCode())
	}

	if mock.BaseURL() != "http://localhost:8080" {
		t.Fatalf("expected base URL with port, got %q", mock.BaseURL())
	}

	if mock.EffectiveMethod() != "GET" {
		t.Fatalf("expected default method GET, got %q", mock.EffectiveMethod())
	}

	if mock.EffectiveCommunicationType() != Internal {
		t.Fatalf("expected default communication type Internal, got %d", mock.EffectiveCommunicationType())
	}

	if mock.ExposesHTTP() {
		t.Fatal("expected default communication type to avoid HTTP exposure")
	}
}

func TestAddResponseRejectsDuplicateID(t *testing.T) {
	t.Parallel()

	mock, err := NewHttpMockModel("http://localhost", 8080, Response{ID: "same"})
	if err != nil {
		t.Fatalf("expected valid mock: %v", err)
	}

	err = mock.AddResponse(Response{ID: "same"})
	if err == nil {
		t.Fatal("expected duplicate response error")
	}
}

func TestResponseByIDReturnsClone(t *testing.T) {
	t.Parallel()

	mock, err := NewHttpMockModel("http://localhost", 8080, Response{
		ID:      "success",
		Headers: map[string]string{"X-Test": "original"},
		Body:    []byte("original"),
	})
	if err != nil {
		t.Fatalf("expected valid mock: %v", err)
	}

	response, found := mock.ResponseByID("success")
	if !found {
		t.Fatal("expected response to be found")
	}

	response.Headers["X-Test"] = "changed"
	response.Body[0] = 'X'

	unchanged, found := mock.ResponseByID("success")
	if !found {
		t.Fatal("expected response to still be found")
	}

	if unchanged.Headers["X-Test"] != "original" {
		t.Fatalf("expected header clone, got %q", unchanged.Headers["X-Test"])
	}

	if string(unchanged.Body) != "original" {
		t.Fatalf("expected body clone, got %q", string(unchanged.Body))
	}
}

func TestAddPayloadRequiresKnownResponse(t *testing.T) {
	t.Parallel()

	mock, err := NewHttpMockModel("http://localhost", 8080, Response{ID: "success"})
	if err != nil {
		t.Fatalf("expected valid mock: %v", err)
	}

	if err := mock.AddPayload(Payload{ResponseID: "missing"}); err == nil {
		t.Fatal("expected unknown response error")
	}

	if err := mock.AddPayload(Payload{ResponseID: "success", Body: []byte("request")}); err != nil {
		t.Fatalf("expected payload to be valid: %v", err)
	}
}

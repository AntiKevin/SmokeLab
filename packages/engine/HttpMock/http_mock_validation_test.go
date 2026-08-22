// Nome: http_mock_validation_test.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Exercita as regras de validação dos mocks HTTP, cobrindo modelos válidos
// e inválidos para proteger as invariantes do domínio e impedir que mudanças futuras
// aceitem configurações inconsistentes.
package HttpMock

import "testing"

func TestNewHttpMockModelValidatesURLAndPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		rawURL string
		port   int
	}{
		{name: "empty URL", rawURL: "", port: 8080},
		{name: "relative URL", rawURL: "localhost", port: 8080},
		{name: "zero port", rawURL: "http://localhost", port: 0},
		{name: "port too high", rawURL: "http://localhost", port: 65536},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := NewHttpMockModel(tt.rawURL, tt.port); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestResponseValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response Response
		wantErr  bool
	}{
		{name: "empty ID", response: Response{}, wantErr: true},
		{name: "default status", response: Response{ID: "ok"}, wantErr: false},
		{name: "lower than HTTP range", response: Response{ID: "bad", StatusCode: 99}, wantErr: true},
		{name: "higher than HTTP range", response: Response{ID: "bad", StatusCode: 600}, wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.response.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHttpMockModelMethodValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		method  string
		wantErr bool
	}{
		{name: "default method", method: "", wantErr: false},
		{name: "lowercase method", method: "post", wantErr: false},
		{name: "invalid method", method: "FETCH", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := HttpMockModel{
				URL:    "http://localhost",
				Port:   8080,
				Method: tt.method,
			}

			err := mock.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHttpMockModelCommunicationTypeValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		communicationType CommunicationType
		wantErr           bool
		exposesHTTP       bool
	}{
		{name: "internal", communicationType: Internal, wantErr: false, exposesHTTP: false},
		{name: "external", communicationType: External, wantErr: false, exposesHTTP: true},
		{name: "both", communicationType: Both, wantErr: false, exposesHTTP: true},
		{name: "invalid", communicationType: CommunicationType(99), wantErr: true, exposesHTTP: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := HttpMockModel{
				URL:               "http://localhost",
				Port:              8080,
				CommunicationType: tt.communicationType,
			}

			err := mock.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if mock.ExposesHTTP() != tt.exposesHTTP {
				t.Fatalf("ExposesHTTP() = %v, want %v", mock.ExposesHTTP(), tt.exposesHTTP)
			}
		})
	}
}

func TestPayloadValidation(t *testing.T) {
	t.Parallel()

	if err := (Payload{}).Validate(); err == nil {
		t.Fatal("expected empty response ID to be invalid")
	}

	if err := (Payload{ResponseID: "success"}).Validate(); err != nil {
		t.Fatalf("expected payload to be valid: %v", err)
	}
}

// Nome: http_mock_validation.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Concentra as regras de validação dos modelos de mock HTTP, verificando
// consistência de IDs, rotas, payloads, códigos de resposta e configurações antes
// que o engine permita simulação ou execução.
package HttpMock

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Validate checks if the mock service definition is ready to run.
func (m HttpMockModel) Validate() error {
	if strings.TrimSpace(m.URL) == "" {
		return errors.New("mock URL is required")
	}

	parsedURL, err := url.ParseRequestURI(m.URL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("invalid mock URL %q", m.URL)
	}

	if m.Port < 1 || m.Port > 65535 {
		return fmt.Errorf("mock port must be between 1 and 65535: %d", m.Port)
	}

	if !isSupportedHTTPMethod(m.EffectiveMethod()) {
		return fmt.Errorf("mock has invalid HTTP method: %q", m.Method)
	}

	if !isSupportedCommunicationType(m.EffectiveCommunicationType()) {
		return fmt.Errorf("mock has invalid communication type: %d", m.CommunicationType)
	}

	seenResponses := make(map[string]struct{}, len(m.Responses))
	for _, response := range m.Responses {
		if err := response.Validate(); err != nil {
			return err
		}

		if _, found := seenResponses[response.ID]; found {
			return fmt.Errorf("response %q already exists", response.ID)
		}
		seenResponses[response.ID] = struct{}{}
	}

	for _, payload := range m.Payloads {
		if err := payload.Validate(); err != nil {
			return err
		}

		if _, found := seenResponses[payload.ResponseID]; !found {
			return fmt.Errorf("payload references unknown response %q", payload.ResponseID)
		}
	}

	return nil
}

// Validate checks if the response can be served by an HTTP mock.
func (r Response) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return errors.New("response ID is required")
	}

	if r.StatusCode == 0 {
		return nil
	}

	if r.StatusCode < 100 || r.StatusCode > 599 {
		return fmt.Errorf("response %q has invalid HTTP status code: %d", r.ID, r.StatusCode)
	}

	return nil
}

// Validate checks if the payload references a response.
func (p Payload) Validate() error {
	if strings.TrimSpace(p.ResponseID) == "" {
		return errors.New("payload response ID is required")
	}

	return nil
}

func isSupportedHTTPMethod(method string) bool {
	switch method {
	case http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace:
		return true
	default:
		return false
	}
}

func isSupportedCommunicationType(communicationType CommunicationType) bool {
	switch communicationType {
	case Internal, External, Both:
		return true
	default:
		return false
	}
}

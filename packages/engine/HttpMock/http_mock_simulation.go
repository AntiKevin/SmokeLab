// Nome: http_mock_simulation.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Implementa a simulação de mocks HTTP no engine, resolvendo requisições
// contra rotas, payloads e respostas configuradas para prever o resultado sem
// depender de uma interface específica ou de execução externa.
package HttpMock

import (
	"errors"
	"fmt"
	"strings"
)

var (
	errMockRequestHeaderMismatch = errors.New("mock request header does not match")
	errMockRequestMethodMismatch = errors.New("mock request method does not match")
	errMockRequestPayloadMissing = errors.New("mock request payload does not match")
)

// Simulate resolves a request in memory, without HTTP communication.
func (m HttpMockModel) Simulate(request MockRequest) (Response, error) {
	if err := m.Validate(); err != nil {
		return Response{}, err
	}

	return m.responseForRequest(request)
}

func (m HttpMockModel) responseForRequest(request MockRequest) (Response, error) {
	if effectiveMockRequestMethod(request.Method, m.EffectiveMethod()) != m.EffectiveMethod() {
		return Response{}, errMockRequestMethodMismatch
	}

	for key, value := range m.Headers {
		if headerValue(request.Headers, key) != value {
			return Response{}, errMockRequestHeaderMismatch
		}
	}

	if len(m.Payloads) == 0 {
		if len(m.Responses) == 0 {
			return Response{}, nil
		}

		return m.Responses[0].Clone(), nil
	}

	for _, payload := range m.Payloads {
		if string(payload.Body) != string(request.Body) {
			continue
		}

		response, found := m.ResponseByID(payload.ResponseID)
		if !found {
			return Response{}, fmt.Errorf("payload references unknown response %q", payload.ResponseID)
		}

		return response, nil
	}

	return Response{}, errMockRequestPayloadMissing
}

func effectiveMockRequestMethod(method string, defaultMethod string) string {
	method = strings.TrimSpace(method)
	if method == "" {
		return defaultMethod
	}

	return strings.ToUpper(method)
}

func headerValue(headers map[string]string, key string) string {
	if len(headers) == 0 {
		return ""
	}

	if value, found := headers[key]; found {
		return value
	}

	for headerKey, value := range headers {
		if strings.EqualFold(headerKey, key) {
			return value
		}
	}

	return ""
}

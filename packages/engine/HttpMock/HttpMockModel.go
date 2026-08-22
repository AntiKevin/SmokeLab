// Nome: HttpMockModel.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Define o modelo de domínio dos mocks HTTP no engine, incluindo respostas,
// payloads, rotas, cenários e tipos de comunicação usados para representar o
// comportamento esperado antes da validação e execução.
package HttpMock

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const defaultMockStatusCode = 200

// Response describes the HTTP response that a mock can return.
type Response struct {
	ID         string            `json:"id"`
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       []byte            `json:"body,omitempty"`
}

// Payload points a request payload to the response that should be used.
type Payload struct {
	ResponseID string `json:"responseId"`
	Body       []byte `json:"body,omitempty"`
}

// CommunicationType defines how the mock can be reached when it executes.
type CommunicationType int

const (
	Internal CommunicationType = iota
	External
	Both
)

// MockRequest describes an in-app request that can be resolved without HTTP.
type MockRequest struct {
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    []byte            `json:"body,omitempty"`
}

// HttpMockModel describes one HTTP mock service.
type HttpMockModel struct {
	URL               string            `json:"url"`
	Port              int               `json:"port"`
	Headers           map[string]string `json:"headers,omitempty"`
	Method            string            `json:"method,omitempty"`
	Payloads          []Payload         `json:"payloads"`
	Responses         []Response        `json:"responses,omitempty"`
	CommunicationType CommunicationType `json:"communicationType"`
}

// NewHttpMockModel creates a validated mock service definition.
func NewHttpMockModel(rawURL string, port int, responses ...Response) (*HttpMockModel, error) {
	mock := &HttpMockModel{
		URL:       strings.TrimSpace(rawURL),
		Port:      port,
		Payloads:  make([]Payload, 0),
		Responses: make([]Response, 0, len(responses)),
	}

	for _, response := range responses {
		if err := mock.AddResponse(response); err != nil {
			return nil, err
		}
	}

	if err := mock.Validate(); err != nil {
		return nil, err
	}

	return mock, nil
}

// AddResponse registers a response, keeping the model immutable from callers.
func (m *HttpMockModel) AddResponse(response Response) error {
	if err := response.Validate(); err != nil {
		return err
	}

	if _, found := m.ResponseByID(response.ID); found {
		return fmt.Errorf("response %q already exists", response.ID)
	}

	m.Responses = append(m.Responses, response.Clone())
	return nil
}

// AddPayload registers a payload, keeping the model immutable from callers.
func (m *HttpMockModel) AddPayload(payload Payload) error {
	if err := payload.Validate(); err != nil {
		return err
	}

	if _, found := m.ResponseByID(payload.ResponseID); !found {
		return fmt.Errorf("payload references unknown response %q", payload.ResponseID)
	}

	m.Payloads = append(m.Payloads, payload.Clone())
	return nil
}

// ResponseByID finds a response and returns a copy that callers can mutate.
func (m HttpMockModel) ResponseByID(id string) (Response, bool) {
	for _, response := range m.Responses {
		if response.ID == id {
			return response.Clone(), true
		}
	}

	return Response{}, false
}

// BaseURL returns the mock service URL with the configured port applied.
func (m HttpMockModel) BaseURL() string {
	parsedURL, err := url.Parse(m.URL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return ""
	}

	parsedURL.Host = net.JoinHostPort(parsedURL.Hostname(), strconv.Itoa(m.Port))
	return strings.TrimRight(parsedURL.String(), "/")
}

// EffectiveMethod returns the configured HTTP method or the default method.
func (m HttpMockModel) EffectiveMethod() string {
	method := strings.TrimSpace(m.Method)
	if method == "" {
		return http.MethodGet
	}

	return strings.ToUpper(method)
}

// EffectiveCommunicationType returns the configured communication type.
func (m HttpMockModel) EffectiveCommunicationType() CommunicationType {
	return m.CommunicationType
}

// ExposesHTTP reports whether Execute will reserve a real HTTP port.
func (m HttpMockModel) ExposesHTTP() bool {
	switch m.EffectiveCommunicationType() {
	case External, Both:
		return true
	default:
		return false
	}
}

func (m HttpMockModel) clone() HttpMockModel {
	clone := HttpMockModel{
		URL:               m.URL,
		Port:              m.Port,
		Method:            m.Method,
		CommunicationType: m.CommunicationType,
		Payloads:          make([]Payload, 0, len(m.Payloads)),
		Responses:         make([]Response, 0, len(m.Responses)),
	}

	if len(m.Headers) > 0 {
		clone.Headers = make(map[string]string, len(m.Headers))
		for key, value := range m.Headers {
			clone.Headers[key] = value
		}
	}

	for _, payload := range m.Payloads {
		clone.Payloads = append(clone.Payloads, payload.Clone())
	}

	for _, response := range m.Responses {
		clone.Responses = append(clone.Responses, response.Clone())
	}

	return clone
}

// Clone returns a deep copy of the response body and headers.
func (r Response) Clone() Response {
	clone := Response{
		ID:         r.ID,
		StatusCode: r.StatusCode,
		Body:       append([]byte(nil), r.Body...),
	}

	if len(r.Headers) > 0 {
		clone.Headers = make(map[string]string, len(r.Headers))
		for key, value := range r.Headers {
			clone.Headers[key] = value
		}
	}

	return clone
}

// EffectiveStatusCode returns the configured status code or the default code.
func (r Response) EffectiveStatusCode() int {
	if r.StatusCode == 0 {
		return defaultMockStatusCode
	}

	return r.StatusCode
}

// Clone returns a copy of the payload body.
func (p Payload) Clone() Payload {
	return Payload{
		ResponseID: p.ResponseID,
		Body:       append([]byte(nil), p.Body...),
	}
}

// Nome: http_mock_execution.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Define estruturas e fluxos de execução dos mocks HTTP no engine,
// preparando o comportamento validado para uso operacional e mantendo a regra
// independente de detalhes de GUI ou CLI.
package HttpMock

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
)

// HttpMockExecution represents a running mock service.
type HttpMockExecution struct {
	model   HttpMockModel
	baseURL string
	server  *http.Server
	done    chan error
}

// Execute starts the mock. External mocks expose HTTP; internal mocks do not.
func (m HttpMockModel) Execute() (*HttpMockExecution, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}

	runningModel := m.clone()
	execution := &HttpMockExecution{
		model:   runningModel,
		baseURL: runningModel.BaseURL(),
		done:    make(chan error, 1),
	}

	if !runningModel.ExposesHTTP() {
		execution.done <- nil
		close(execution.done)
		return execution, nil
	}

	parsedURL, err := url.Parse(runningModel.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid mock URL %q: %w", runningModel.URL, err)
	}

	address := net.JoinHostPort(parsedURL.Hostname(), strconv.Itoa(runningModel.Port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("start mock service on %s: %w", address, err)
	}

	server := &http.Server{
		Handler: runningModel.httpHandler(parsedURL.EscapedPath()),
	}
	execution.server = server

	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		execution.done <- err
		close(execution.done)
	}()

	return execution, nil
}

// BaseURL returns the configured HTTP URL for the mock.
func (e *HttpMockExecution) BaseURL() string {
	if e == nil {
		return ""
	}

	return e.baseURL
}

// Done is closed when the execution stops.
func (e *HttpMockExecution) Done() <-chan error {
	if e == nil {
		done := make(chan error)
		close(done)
		return done
	}

	return e.done
}

// Shutdown stops an external HTTP execution. Internal executions are no-ops.
func (e *HttpMockExecution) Shutdown(ctx context.Context) error {
	if e == nil || e.server == nil {
		return nil
	}

	return e.server.Shutdown(ctx)
}

// Simulate resolves a request in memory, without HTTP communication.
func (e *HttpMockExecution) Simulate(request MockRequest) (Response, error) {
	if e == nil {
		return Response{}, errors.New("mock execution is nil")
	}

	return e.model.Simulate(request)
}

func (m HttpMockModel) httpHandler(path string) http.Handler {
	if path == "" {
		path = "/"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != path {
			http.NotFound(w, r)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read mock request body", http.StatusBadRequest)
			return
		}

		response, err := m.responseForRequest(MockRequest{
			Method:  r.Method,
			Headers: headersFromHTTP(r.Header),
			Body:    body,
		})
		if err != nil {
			writeMockRequestError(w, err)
			return
		}

		for key, value := range response.Headers {
			w.Header().Set(key, value)
		}

		w.WriteHeader(response.EffectiveStatusCode())
		_, _ = w.Write(response.Body)
	})
}

func headersFromHTTP(headers http.Header) map[string]string {
	if len(headers) == 0 {
		return nil
	}

	result := make(map[string]string, len(headers))
	for key, values := range headers {
		if len(values) == 0 {
			continue
		}

		result[key] = values[0]
	}

	return result
}

func writeMockRequestError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errMockRequestMethodMismatch):
		http.Error(w, err.Error(), http.StatusMethodNotAllowed)
	case errors.Is(err, errMockRequestHeaderMismatch):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, errMockRequestPayloadMissing):
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

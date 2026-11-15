package barlivetest

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"

	"github.com/backbone81/bar-replay-analytics/internal/barlive"
)

// StoragePath is the path under which the clients of a Server find the replay storage.
const StoragePath = "/storage"

// Request is a request as a Server received it.
type Request struct {
	// Path is the escaped path, as the client sent it.
	Path   string
	Query  url.Values
	Header http.Header
}

// Server is a local HTTP server for Teiserver, bar-db and the replay storage together. It answers with the handlers
// registered on its ServeMux, and 404 for any other request, and records every request it receives.
type Server struct {
	*http.ServeMux

	server *httptest.Server

	mutex    sync.Mutex
	requests []Request
}

// NewServer starts a Server. Close it when done.
func NewServer() *Server {
	server := &Server{ServeMux: http.NewServeMux()}
	server.server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		server.mutex.Lock()
		server.requests = append(server.requests, Request{
			Path:   request.URL.EscapedPath(),
			Query:  request.URL.Query(),
			Header: request.Header.Clone(),
		})
		server.mutex.Unlock()
		server.ServeHTTP(writer, request)
	}))
	return server
}

// Close shuts the server down.
func (s *Server) Close() {
	s.server.Close()
}

// Client returns a client which sends every request to the server, without a rate limit unless options set one, and
// fails any request to another host, so no test can reach the real services. options cannot redirect the client.
func (s *Server) Client(options ...barlive.Option) *barlive.Client {
	options = append([]barlive.Option{barlive.WithMaxRequestRate(0)}, options...)
	return barlive.NewClient(slog.New(slog.DiscardHandler), append(options,
		barlive.WithTransport(&localTransport{
			host: s.server.Listener.Addr().String(),
			base: s.server.Client().Transport,
		}),
		barlive.WithTeiserverURL(s.server.URL),
		barlive.WithReplayAPIURL(s.server.URL),
		barlive.WithStorageURL(s.server.URL+StoragePath),
	)...)
}

// Requests returns the requests received so far.
func (s *Server) Requests() []Request {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return append([]Request(nil), s.requests...)
}

// RequestsTo returns the requests received so far for the escaped path.
func (s *Server) RequestsTo(path string) []Request {
	var requests []Request
	for _, request := range s.Requests() {
		if request.Path == path {
			requests = append(requests, request)
		}
	}
	return requests
}

// localTransport sends requests to host only.
type localTransport struct {
	host string
	base http.RoundTripper
}

func (t *localTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.URL.Host != t.host {
		return nil, fmt.Errorf("a test must not reach %s", request.URL.Host)
	}
	return t.base.RoundTrip(request)
}

// ServeString answers with status and body of contentType.
func ServeString(status int, contentType string, body string) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", contentType)
		writer.WriteHeader(status)
		_, _ = writer.Write([]byte(body))
	}
}

// ServeJSON answers with status and value as JSON, as Teiserver and bar-db do. If value cannot be encoded, it answers
// with 500 Internal Server Error.
func ServeJSON(status int, value any) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		data, err := json.Marshal(value)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		ServeString(status, "application/json; charset=utf-8", string(data))(writer, request)
	}
}

// ServeJSONFile answers with status and the content of the JSON file filePath, such as a response recorded from the
// real service. If the file cannot be read, it answers with 500 Internal Server Error.
func ServeJSONFile(status int, filePath string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		data, err := os.ReadFile(filePath) //nolint:gosec // a file of the test
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		ServeString(status, "application/json; charset=utf-8", string(data))(writer, request)
	}
}

// ServeDemo answers like the replay storage with the demo file content.
func ServeDemo(content []byte) http.HandlerFunc {
	// The storage really sends this content type.
	return ServeString(http.StatusOK, "application/x-www-form-urlencoded", string(content))
}

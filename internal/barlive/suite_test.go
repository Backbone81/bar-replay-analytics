package barlive_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/barlive"
)

func TestBarlive(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Barlive Suite")
}

// testStoragePath is where the test server serves the replay storage.
const testStoragePath = "/storage"

// testServer stands in for Teiserver, bar-db and the replay storage, and records every request it receives.
type testServer struct {
	*http.ServeMux

	server *httptest.Server

	mutex    sync.Mutex
	requests []recordedRequest
}

// recordedRequest is a request as the test server received it.
type recordedRequest struct {
	Time   time.Time
	Path   string
	Query  url.Values
	Header http.Header
}

// newTestServer starts a test server which the spec closes when it ends.
func newTestServer() *testServer {
	testServer := &testServer{ServeMux: http.NewServeMux()}
	testServer.server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		testServer.mutex.Lock()
		testServer.requests = append(testServer.requests, recordedRequest{
			Time:   time.Now(),
			Path:   request.URL.EscapedPath(),
			Query:  request.URL.Query(),
			Header: request.Header.Clone(),
		})
		testServer.mutex.Unlock()
		testServer.ServeHTTP(writer, request)
	}))
	DeferCleanup(testServer.server.Close)
	return testServer
}

// client returns a client which sends all requests to the test server, and fails any request to another host, so no
// spec can reach the real services. The rate limit is high, so specs do not wait, unless options set another. The
// options cannot redirect the client.
func (s *testServer) client(options ...barlive.Option) *barlive.Client {
	serverURL, err := url.Parse(s.server.URL)
	Expect(err).NotTo(HaveOccurred())

	options = append([]barlive.Option{barlive.WithMaxRequestRate(1000)}, options...)
	return barlive.NewClient(append(options,
		barlive.WithTransport(&testTransport{
			host: serverURL.Host,
			base: s.server.Client().Transport,
		}),
		barlive.WithTeiserverURL(s.server.URL),
		barlive.WithReplayAPIURL(s.server.URL),
		barlive.WithStorageURL(s.server.URL+testStoragePath),
	)...)
}

// recorded returns the requests received so far.
func (s *testServer) recorded() []recordedRequest {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return append([]recordedRequest(nil), s.requests...)
}

// testTransport sends requests to host only.
type testTransport struct {
	host string
	base http.RoundTripper
}

func (t *testTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.URL.Host != t.host {
		return nil, fmt.Errorf("a test must not reach %s", request.URL.Host)
	}
	return t.base.RoundTrip(request)
}

// serveTestdata answers with the content of testdata/<name> and status.
func serveTestdata(status int, name string) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		Expect(err).NotTo(HaveOccurred())
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		writer.WriteHeader(status)
		_, _ = writer.Write(data)
	}
}

// serveString answers with body and status.
func serveString(status int, contentType string, body string) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", contentType)
		writer.WriteHeader(status)
		_, _ = writer.Write([]byte(body))
	}
}

// requestRecorder records the URLs Config.OnRequest reports, with the time of each.
type requestRecorder struct {
	mutex sync.Mutex
	urls  []string
	times []time.Time
}

// record is the Config.OnRequest of the recorder.
func (r *requestRecorder) record(requestURL string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.urls = append(r.urls, requestURL)
	r.times = append(r.times, time.Now())
}

// recordedURLs returns the URLs reported so far.
func (r *requestRecorder) recordedURLs() []string {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return append([]string(nil), r.urls...)
}

// recordedTimes returns the times of the reports so far.
func (r *requestRecorder) recordedTimes() []time.Time {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return append([]time.Time(nil), r.times...)
}

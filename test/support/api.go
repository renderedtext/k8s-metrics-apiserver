package testsupport

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/semaphoreci/k8s-metrics-apiserver/pkg/common"
)

type APIMockServer struct {
	Server     *httptest.Server
	Handler    http.Handler
	AgentTypes map[string]common.Metrics
}

func NewAPIMockServer() *APIMockServer {
	return &APIMockServer{
		AgentTypes: map[string]common.Metrics{},
	}
}

func (m *APIMockServer) Init() {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/api/v1/self_hosted_agents/metrics") {
			m.handleRequest(w, r)
		} else {
			w.WriteHeader(404)
		}
	}))

	m.Server = mockServer
	fmt.Printf("Started Semaphore API mock at %s\n", mockServer.URL)
}

func (m *APIMockServer) RegisterAgentType(token string, metrics common.Metrics) {
	m.AgentTypes[token] = metrics
}

func (m *APIMockServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	token := strings.Replace(r.Header.Get("Authorization"), "Token ", "", 1)
	fmt.Printf("[Semaphore API mock] Received request with token %s", token)

	metrics, exists := m.AgentTypes[token]
	if !exists {
		fmt.Printf("[Semaphore API mock] Agent type with token %s is not registered\n", token)
		w.WriteHeader(500)
		return
	}

	data, err := json.Marshal(&metrics)
	if err != nil {
		fmt.Printf("[Semaphore API mock] Error marshaling response: %v\n", err)
		w.WriteHeader(500)
		return
	}

	_, _ = w.Write(data)
}

func (m *APIMockServer) URL() string {
	return m.Server.URL
}

func (m *APIMockServer) Host() string {
	return m.Server.Listener.Addr().String()
}

func (m *APIMockServer) Close() {
	m.Server.Close()
}

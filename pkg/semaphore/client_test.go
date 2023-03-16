package semaphore

import (
	"net/http"
	"testing"

	"github.com/semaphoreci/k8s-metrics-apiserver/pkg/common"
	testsupport "github.com/semaphoreci/k8s-metrics-apiserver/test/support"
	"github.com/stretchr/testify/assert"
)

func Test__GetMetricsForSingleAgentType(t *testing.T) {
	apiMock := testsupport.NewAPIMockServer()
	apiMock.Init()

	m1 := common.Metrics{
		Agents: common.AgentMetrics{Idle: 0, Occupied: 10},
		Jobs:   common.JobMetrics{Running: 10, Queued: 10},
	}
	apiMock.RegisterAgentType("agent-type-1-token", m1)

	c := NewClient(http.DefaultClient, true)
	metrics := c.GetMetrics([]*common.AgentType{
		{
			Name:     "agent-type-1",
			Endpoint: apiMock.Host(),
			Token:    "agent-type-1-token",
		},
	})

	expected := m1.GenerateAll(map[string]string{"agent_type": "agent-type-1"})
	for i, v := range metrics {
		assert.Equal(t, v.MetricLabels, expected[i].MetricLabels)
		assert.Equal(t, v.MetricName, expected[i].MetricName)
		assert.Equal(t, v.Value, expected[i].Value)
	}

	apiMock.Close()
}

func Test__GetMetricsForMultipleAgentTypes(t *testing.T) {
	apiMock := testsupport.NewAPIMockServer()
	apiMock.Init()

	m1 := common.Metrics{
		Agents: common.AgentMetrics{Idle: 0, Occupied: 10},
		Jobs:   common.JobMetrics{Running: 10, Queued: 10},
	}

	m2 := common.Metrics{
		Agents: common.AgentMetrics{Idle: 5, Occupied: 5},
		Jobs:   common.JobMetrics{Running: 5, Queued: 0},
	}

	apiMock.RegisterAgentType("agent-type-1-token", m1)
	apiMock.RegisterAgentType("agent-type-2-token", m2)

	c := NewClient(http.DefaultClient, true)
	metrics := c.GetMetrics([]*common.AgentType{
		{
			Name:     "agent-type-1",
			Endpoint: apiMock.Host(),
			Token:    "agent-type-1-token",
		},
		{
			Name:     "agent-type-2",
			Endpoint: apiMock.Host(),
			Token:    "agent-type-2-token",
		},
	})

	expected := append(
		m1.GenerateAll(map[string]string{"agent_type": "agent-type-1"}),
		m2.GenerateAll(map[string]string{"agent_type": "agent-type-2"})...,
	)

	for i, v := range metrics {
		assert.Equal(t, v.MetricLabels, expected[i].MetricLabels)
		assert.Equal(t, v.MetricName, expected[i].MetricName)
		assert.Equal(t, v.Value, expected[i].Value)
	}
}

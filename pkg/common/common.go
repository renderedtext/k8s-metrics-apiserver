package common

import (
	"fmt"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const (
	MetricAgentsTotal              = "agents_total"
	MetricAgentsIdle               = "agents_idle"
	MetricAgentsOccupied           = "agents_occupied"
	MetricAgentsOccupiedPercentage = "agents_occupied_percentage"
	MetricJobsTotal                = "jobs_total"
	MetricJobsQueued               = "jobs_queued"
	MetricJobsRunning              = "jobs_running"
)

type AgentType struct {
	Name     string
	Endpoint string
	Token    string
}

var AllMetrics = []string{
	MetricAgentsTotal,
	MetricAgentsIdle,
	MetricAgentsOccupied,
	MetricAgentsOccupiedPercentage,
	MetricJobsTotal,
	MetricJobsQueued,
	MetricJobsRunning,
}

type Metrics struct {
	Jobs   JobMetrics
	Agents AgentMetrics
}

func (m *Metrics) GenerateAll(labels map[string]string) []external_metrics.ExternalMetricValue {
	values := []external_metrics.ExternalMetricValue{}

	for _, metricName := range AllMetrics {
		values = append(values, external_metrics.ExternalMetricValue{
			MetricName:   metricName,
			Timestamp:    v1.NewTime(time.Now()),
			Value:        resource.MustParse(m.Calc(metricName)),
			MetricLabels: labels,
		})
	}

	return values
}

func (m *Metrics) Calc(metricName string) string {
	switch metricName {
	case MetricAgentsTotal:
		return strconv.Itoa(m.Agents.Total())
	case MetricAgentsIdle:
		return strconv.Itoa(m.Agents.Idle)
	case MetricAgentsOccupied:
		return strconv.Itoa(m.Agents.Occupied)
	case MetricAgentsOccupiedPercentage:
		return strconv.Itoa(m.Agents.OccupiedPercentage())
	case MetricJobsTotal:
		return strconv.Itoa(m.Jobs.Total())
	case MetricJobsQueued:
		return strconv.Itoa(m.Jobs.Queued)
	case MetricJobsRunning:
		return strconv.Itoa(m.Jobs.Running)
	default:
		return ""
	}
}

func (m *Metrics) String() string {
	return fmt.Sprintf(
		"agents/occupied=%d agents/idle=%d jobs/queued=%d jobs/running=%d",
		m.Agents.Occupied,
		m.Agents.Idle,
		m.Jobs.Queued,
		m.Jobs.Running,
	)
}

type JobMetrics struct {
	Queued  int
	Running int
}

func (m *JobMetrics) Total() int {
	return m.Queued + m.Running
}

type AgentMetrics struct {
	Idle     int
	Occupied int
}

func (m *AgentMetrics) Total() int {
	return m.Idle + m.Occupied
}

func (m *AgentMetrics) OccupiedPercentage() int {
	if m.Total() > 0 {
		return 100 * (m.Occupied / m.Total())
	}

	// TODO: not sure we should return 0 here
	return 0
}

package semaphore

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/semaphoreci/k8s-metrics-apiserver/pkg/common"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
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

var AllMetrics = []string{
	MetricAgentsTotal,
	MetricAgentsIdle,
	MetricAgentsOccupied,
	MetricAgentsOccupiedPercentage,
	MetricJobsTotal,
	MetricJobsQueued,
	MetricJobsRunning,
}

type Client struct {
	httpClient *http.Client
}

func NewClient(httpClient *http.Client) *Client {
	return &Client{httpClient: httpClient}
}

type Metrics struct {
	Jobs   JobMetrics
	Agents AgentMetrics
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

func (c *Client) GetMetrics(agentTypes []*common.AgentType) []external_metrics.ExternalMetricValue {
	values := []external_metrics.ExternalMetricValue{}

	for _, agentType := range agentTypes {
		m, err := c.getForAgentType(agentType.Endpoint, agentType.Token)
		if err != nil {
			klog.Errorf("Error collecting metrics from Semaphore API for %s: %v", agentType.Name, err)
			continue
		}

		klog.Infof("Metrics for %s: %s", agentType.Name, m.String())

		// For each metric we should export, store it in our map
		for _, metricName := range AllMetrics {
			values = append(values, external_metrics.ExternalMetricValue{
				MetricName: metricName,
				Timestamp:  v1.NewTime(time.Now()),
				Value:      resource.MustParse(c.calc(m, metricName)),
				MetricLabels: map[string]string{
					"agent_type": agentType.Name,
				},
			})
		}
	}

	return values
}

func (c *Client) getForAgentType(endpoint, token string) (*Metrics, error) {
	url := fmt.Sprintf("https://%s/api/v1/self_hosted_agents/metrics", endpoint)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", token))
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("request failed with %d", res.StatusCode)
	}

	response, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var m Metrics
	err = json.Unmarshal(response, &m)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &m, nil
}

func (c *Client) calc(m *Metrics, metricName string) string {
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

package semaphore

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

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

func (c *Client) GetMetrics(endpoint, token string) (*Metrics, error) {
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

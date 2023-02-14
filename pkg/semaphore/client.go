package semaphore

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Client struct {
	httpClient *http.Client
	endpoint   string
	token      string
}

func NewClient(httpClient *http.Client, endpoint, token string) *Client {
	return &Client{
		httpClient: httpClient,
		endpoint:   endpoint,
		token:      token,
	}
}

type Metrics struct {
	Jobs   JobMetrics
	Agents AgentMetrics
}

type JobMetrics struct {
	Queued  int
	Running int
}

type AgentMetrics struct {
	Idle     int
	Occupied int
}

func (c *Client) GetMetrics() (*Metrics, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s/api/v1/self_hosted_agents/metrics", c.endpoint), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", c.token))
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

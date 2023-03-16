package semaphore

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/semaphoreci/k8s-metrics-apiserver/pkg/common"
	"k8s.io/klog"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type Client struct {
	httpClient *http.Client
	useHTTP    bool
}

func NewClient(httpClient *http.Client, useHTTP bool) *Client {
	return &Client{httpClient: httpClient, useHTTP: useHTTP}
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
		labels := map[string]string{"agent_type": agentType.Name}
		values = append(values, m.GenerateAll(labels)...)
	}

	return values
}

func (c *Client) getURL(endpoint string) string {
	if c.useHTTP {
		return fmt.Sprintf("http://%s/api/v1/self_hosted_agents/metrics", endpoint)
	}

	return fmt.Sprintf("https://%s/api/v1/self_hosted_agents/metrics", endpoint)
}

func (c *Client) getForAgentType(endpoint, token string) (*common.Metrics, error) {
	req, err := http.NewRequest("GET", c.getURL(endpoint), nil)
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

	var m common.Metrics
	err = json.Unmarshal(response, &m)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &m, nil
}

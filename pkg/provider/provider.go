package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog"
	metrics "k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"

	"github.com/dgraph-io/ristretto"
	"github.com/semaphoreci/k8s-metrics-apiserver/pkg/semaphore"
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

// We cache the agent type secret information
// to avoid going to the Kubernetes on every iteration.
// But we should also reach to changes to the agent type secrets,
// so we put an expiration on them.
var SecretCacheTTL = 5 * time.Minute

type AgentTypeInfo struct {
	Name     string
	Endpoint string
	Token    string
}

type SemaphoreMetricsProvider struct {
	secrets     dynamic.ResourceInterface
	secretCache *ristretto.Cache
	config      Config
	data        sync.Map
}

type Config struct {
	Client          dynamic.Interface
	Mapper          apimeta.RESTMapper
	SemaphoreClient *semaphore.Client
}

func New(config Config) (*SemaphoreMetricsProvider, error) {

	namespace := os.Getenv("KUBERNETES_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	// The provider needs read access to secrets,
	// so it can find all the secrets for each agent type,
	// and use the agent type token in them to grab metrics from the Semaphore API.
	c := config.Client.
		Resource(schema.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "secrets",
		}).
		Namespace(namespace)

	/*
	 * We keep at most 50 keys (agent type info) in our cache.
	 */
	secretCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 500,
		MaxCost:     50,
		BufferItems: 64,
		Metrics:     false,
	})

	if err != nil {
		return nil, err
	}

	return &SemaphoreMetricsProvider{
		secrets:     c,
		secretCache: secretCache,
		config:      config,
		data:        sync.Map{},
	}, nil
}

func (p *SemaphoreMetricsProvider) GetExternalMetric(ctx context.Context, namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*metrics.ExternalMetricValueList, error) {
	v, ok := p.data.Load(info.Metric)
	if !ok {
		return &metrics.ExternalMetricValueList{}, nil
	}

	values := v.([]metrics.ExternalMetricValue)

	// If no selector is used, we return metrics for all the agent types
	if metricSelector.Empty() {
		return &metrics.ExternalMetricValueList{
			Items: values,
		}, nil
	}

	// Otherwise we return only the values that match the label selector
	return &metrics.ExternalMetricValueList{
		Items: filterByMetricSelector(values, metricSelector),
	}, nil
}

func (p *SemaphoreMetricsProvider) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	list := []provider.ExternalMetricInfo{}

	p.data.Range(func(key, value any) bool {
		list = append(list, provider.ExternalMetricInfo{Metric: key.(string)})
		return true
	})

	return list
}

// TODO: use noise in intervals
func (p *SemaphoreMetricsProvider) Collect() {
	for {

		// Find all agent type secrets
		list, err := p.secrets.List(context.Background(), v1.ListOptions{
			LabelSelector: "semaphore-agent/autoscaled=true",
		})

		// If we can't list secrets, we try again in the next iteration.
		if err != nil {
			klog.Errorf("Error listing secrets: %v", err)
			time.Sleep(10 * time.Second)
			continue
		}

		// If there are no agent type secrets, we try again in the next iteration.
		if len(list.Items) == 0 {
			klog.Info("No agent type secrets found.")
			time.Sleep(10 * time.Second)
			continue
		}

		// Collect metrics from the Semaphore API
		// for all the agent type secrets found.
		klog.Infof("Found %d agent type secrets", len(list.Items))
		p.collectForAll(list.Items)
		time.Sleep(10 * time.Second)
	}
}

// Collect metrics from the Semaphore API for each agent type secret found.
// We use the /api/v1/self_hosted_agents/metrics endpoint for that.
// That endpoint requires the agent type registration token.
func (p *SemaphoreMetricsProvider) collectForAll(secrets []unstructured.Unstructured) {
	values := []metrics.ExternalMetricValue{}

	// Collect metrics for all agent types
	for _, secret := range secrets {

		// Find agent type information
		agentTypeInfo, err := p.getAgentTypeInfo(secret.GetName())
		if err != nil {
			klog.Errorf("Error finding agent type information from secret '%s': %v", secret.GetName(), err)
			continue
		}

		// Fetch the agent type metrics from the Semaphore API
		m, err := p.config.SemaphoreClient.GetMetrics(agentTypeInfo.Endpoint, agentTypeInfo.Token)
		if err != nil {
			klog.Errorf("Error collecting metrics from Semaphore API for %s: %v", agentTypeInfo.Name, err)
			continue
		}

		klog.Infof("Metrics for %s: %s", agentTypeInfo.Name, m.String())

		// For each metric we should export, store it in our map
		for _, metricName := range AllMetrics {
			values = append(values, metrics.ExternalMetricValue{
				MetricName: metricName,
				Timestamp:  v1.NewTime(time.Now()),
				Value:      resource.MustParse(p.calc(m, metricName)),
				MetricLabels: map[string]string{
					"agent_type": agentTypeInfo.Name,
				},
			})
		}
	}

	// Store them on their proper keys
	for _, metricName := range AllMetrics {
		p.data.Store(metricName, filterByMetricName(values, metricName))
	}
}

func (p *SemaphoreMetricsProvider) calc(m *semaphore.Metrics, metricName string) string {
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

// Get the agent type information (endpoint and token) from the secret specified.
// We also cache this information to avoid going to the Kubernetes API on every iteration.
func (p *SemaphoreMetricsProvider) getAgentTypeInfo(secretName string) (*AgentTypeInfo, error) {
	value, found := p.secretCache.Get(secretName)
	if found {
		if info, ok := value.(*AgentTypeInfo); ok {
			return info, nil
		}
	}

	// If the agent type info does not exist in the cache,
	// we fetch the information from the Kubernetes API.
	o, err := p.secrets.Get(context.Background(), secretName, v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error describing secret: %v", err)
	}

	info, err := p.unstructuredSecretToAgentTypeInfo(o)
	if err != nil {
		return nil, fmt.Errorf("error finding agent type information in secret: %v", err)
	}

	p.secretCache.SetWithTTL(secretName, info, 1, SecretCacheTTL)
	return info, nil
}

func (p *SemaphoreMetricsProvider) unstructuredSecretToAgentTypeInfo(secret *unstructured.Unstructured) (*AgentTypeInfo, error) {
	endpoint, err := getNestedString(secret, "data", "endpoint")
	if err != nil {
		return nil, err
	}

	token, err := getNestedString(secret, "data", "token")
	if err != nil {
		return nil, err
	}

	return &AgentTypeInfo{
		Name:     secret.GetName(),
		Endpoint: endpoint,
		Token:    token,
	}, nil
}

func filterByMetricName(values []metrics.ExternalMetricValue, metricName string) []metrics.ExternalMetricValue {
	filtered := []metrics.ExternalMetricValue{}

	for _, v := range values {
		if v.MetricName == metricName {
			filtered = append(filtered, v)
		}
	}

	return filtered
}

func filterByMetricSelector(values []metrics.ExternalMetricValue, metricSelector labels.Selector) []metrics.ExternalMetricValue {
	filtered := []metrics.ExternalMetricValue{}
	for _, v := range values {
		if metricSelector.Matches(&Labels{metric: v}) {
			filtered = append(filtered, v)
		}
	}

	return filtered
}

func getNestedString(o *unstructured.Unstructured, fields ...string) (string, error) {
	o.GetName()
	v, found, err := unstructured.NestedString(o.Object, fields...)
	if !found || err != nil {
		return "", fmt.Errorf("could not find nested field in unstructured object %v", o.Object)
	}

	decoded, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return "", fmt.Errorf("error decoding nested field from base64: %v", err)
	}

	return string(decoded), nil
}

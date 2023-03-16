package provider

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog"
	metrics "k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"

	"github.com/semaphoreci/k8s-metrics-apiserver/pkg/semaphore"
)

type SemaphoreMetricsProvider struct {
	config Config
	finder *AgentTypeFinder
	data   sync.Map
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

	finder, err := NewAgentTypeFinder(config.Client, namespace)
	if err != nil {
		return nil, fmt.Errorf("error creating agent type finder")
	}

	return &SemaphoreMetricsProvider{
		finder: finder,
		config: config,
		data:   sync.Map{},
	}, nil
}

// Return all metrics in semaphore.AllMetrics
func (p *SemaphoreMetricsProvider) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	list := []provider.ExternalMetricInfo{}

	for _, m := range semaphore.AllMetrics {
		list = append(list, provider.ExternalMetricInfo{Metric: m})
	}

	return list
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

func (p *SemaphoreMetricsProvider) Collect() {
	for {
		agentTypes, err := p.finder.Find()
		if err != nil {
			klog.Errorf("Error finding agent types: %v", err)
		} else {
			klog.Infof("Found %d agent types", len(agentTypes))
			values := p.config.SemaphoreClient.GetMetrics(agentTypes)
			for _, metricName := range semaphore.AllMetrics {
				p.data.Store(metricName, filterByMetricName(values, metricName))
			}
		}

		// TODO: use noise in intervals
		time.Sleep(10 * time.Second)
	}
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

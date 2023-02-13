package provider

import (
	"context"
	"strconv"
	"sync"
	"time"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog"
	metrics "k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"

	"github.com/semaphoreci/k8s-metrics-apiserver/pkg/semaphore"
)

type SemaphoreMetricsProvider struct {
	config Config
	data   sync.Map
}

type Config struct {
	Client          dynamic.Interface
	Mapper          apimeta.RESTMapper
	SemaphoreClient *semaphore.Client
}

func New(config Config) *SemaphoreMetricsProvider {
	return &SemaphoreMetricsProvider{
		config: config,
		data:   sync.Map{},
	}
}

func (p *SemaphoreMetricsProvider) GetExternalMetric(ctx context.Context, namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*metrics.ExternalMetricValueList, error) {
	v, ok := p.data.Load(info.Metric)
	if !ok {
		return &metrics.ExternalMetricValueList{}, nil
	}

	return &metrics.ExternalMetricValueList{
		Items: []metrics.ExternalMetricValue{
			{
				MetricName: info.Metric,
				Timestamp:  v1.NewTime(time.Now()),
				Value:      resource.MustParse(strconv.Itoa(v.(int))),
			},
		},
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
		err := p.collect()
		if err != nil {
			klog.Errorf("error scraping metrics: %s", err)
		}

		time.Sleep(10 * time.Second)
	}
}

func (p *SemaphoreMetricsProvider) collect() error {
	m, err := p.config.SemaphoreClient.GetMetrics()
	if err != nil {
		return err
	}

	p.data.Store("idle_agents", m.Agents.Idle)
	p.data.Store("occupied_agents", m.Agents.Occupied)
	p.data.Store("running_jobs", m.Jobs.Running)
	p.data.Store("queued_jobs", m.Jobs.Queued)

	totalAgents := m.Agents.Idle + m.Agents.Occupied
	if totalAgents > 0 {
		p.data.Store("idle_agents_percentage", m.Agents.Idle/totalAgents)
	}

	return nil
}

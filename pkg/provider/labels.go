package provider

import (
	metrics "k8s.io/metrics/pkg/apis/external_metrics"
)

// Implement of the apimachinery.Labels interface
// See: https://github.com/kubernetes/apimachinery/blob/master/pkg/labels/labels.go
type Labels struct {
	metric metrics.ExternalMetricValue
}

func (l *Labels) Has(label string) (exists bool) {
	for k := range l.metric.MetricLabels {
		if k == label {
			return true
		}
	}
	return false
}

func (l *Labels) Get(label string) (value string) {
	for k, v := range l.metric.MetricLabels {
		if k == label {
			return v
		}
	}

	return ""
}

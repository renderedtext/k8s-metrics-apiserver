package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

func Test__Labels(t *testing.T) {

	t.Run("Has()", func(t *testing.T) {
		l := Labels{metric: external_metrics.ExternalMetricValue{
			MetricLabels: map[string]string{
				"app":   "semaphore-agent",
				"other": "",
			},
		}}

		assert.True(t, l.Has("app"))
		assert.True(t, l.Has("other"))
		assert.False(t, l.Has("app2"))
	})

	t.Run("Get()", func(t *testing.T) {
		l := Labels{metric: external_metrics.ExternalMetricValue{
			MetricLabels: map[string]string{
				"app":   "semaphore-agent",
				"other": "",
			},
		}}

		assert.Equal(t, l.Get("app"), "semaphore-agent")
		assert.Equal(t, l.Get("other"), "")
		assert.Equal(t, l.Get("app2"), "")
	})

	t.Run("Has()", func(t *testing.T) {
		l := Labels{metric: external_metrics.ExternalMetricValue{
			MetricLabels: map[string]string{"app": "semaphore-agent"},
		}}

		assert.True(t, l.Has("app"))
		assert.False(t, l.Has("app2"))
	})

	t.Run("does not match if only key exists", func(t *testing.T) {
		selector := labels.SelectorFromSet(labels.Set(map[string]string{"app": ""}))

		assert.False(t, selector.Matches(&Labels{metric: external_metrics.ExternalMetricValue{
			MetricLabels: map[string]string{"app": "semaphore-agent"},
		}}))
	})

	t.Run("does not match if key exists, but value does not match", func(t *testing.T) {
		selector := labels.SelectorFromSet(labels.Set(map[string]string{"app": "something-else"}))

		assert.False(t, selector.Matches(&Labels{metric: external_metrics.ExternalMetricValue{
			MetricLabels: map[string]string{"app": "semaphore-agent"},
		}}))
	})

	t.Run("does not match if value exists, but key doesn't", func(t *testing.T) {
		selector := labels.SelectorFromSet(labels.Set(map[string]string{"app2": "semaphore-agent"}))

		assert.False(t, selector.Matches(&Labels{metric: external_metrics.ExternalMetricValue{
			MetricLabels: map[string]string{"app": "semaphore-agent"},
		}}))
	})

	t.Run("matches if selector key and value match metric key and value", func(t *testing.T) {
		selector := labels.SelectorFromSet(labels.Set(map[string]string{"app": "semaphore-agent"}))

		assert.True(t, selector.Matches(&Labels{metric: external_metrics.ExternalMetricValue{
			MetricLabels: map[string]string{"app": "semaphore-agent"},
		}}))
	})
}

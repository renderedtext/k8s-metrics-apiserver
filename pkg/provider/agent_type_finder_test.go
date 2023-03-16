package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func Test__AgentTypeFinder(t *testing.T) {
	t.Run("no secrets -> no agent types", func(t *testing.T) {
		c := dynamicfake.NewSimpleDynamicClient(newTestScheme())
		f, _ := NewAgentTypeFinder(c, "default")
		types, err := f.Find()
		assert.NoError(t, err)
		assert.Empty(t, types)
	})

	t.Run("secret exists but has no label -> no agent types", func(t *testing.T) {
		c := dynamicfake.NewSimpleDynamicClient(newTestScheme(), []runtime.Object{
			&corev1.Secret{
				ObjectMeta: v1.ObjectMeta{Name: "agent-type-1", Namespace: "default"},
				Type:       corev1.SecretTypeOpaque,
				Data:       map[string][]byte{},
			},
		}...)

		f, _ := NewAgentTypeFinder(c, "default")
		types, err := f.Find()
		assert.NoError(t, err)
		assert.Empty(t, types)
	})

	t.Run("secret exists but in a different namespace -> no agent types", func(t *testing.T) {
		c := dynamicfake.NewSimpleDynamicClient(newTestScheme(), []runtime.Object{
			&corev1.Secret{
				ObjectMeta: v1.ObjectMeta{Name: "agent-type-1", Namespace: "other"},
				Type:       corev1.SecretTypeOpaque,
				Data:       map[string][]byte{},
			},
		}...)

		f, _ := NewAgentTypeFinder(c, "default")
		types, err := f.Find()
		assert.NoError(t, err)
		assert.Empty(t, types)
	})

	t.Run("secret exists in proper namespace with labels but no keys -> no agent types and error", func(t *testing.T) {
		c := dynamicfake.NewSimpleDynamicClient(newTestScheme(), []runtime.Object{
			&corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      "agent-type-1",
					Namespace: "default",
					Labels:    map[string]string{"semaphore-agent/autoscaled": "true"},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{},
			},
		}...)

		f, _ := NewAgentTypeFinder(c, "default")
		types, err := f.Find()
		assert.Error(t, err)
		assert.Empty(t, types)
	})

	t.Run("secret exists in proper namespace with label -> agent type is returned", func(t *testing.T) {
		c := dynamicfake.NewSimpleDynamicClient(newTestScheme(), []runtime.Object{
			&corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      "agent-type-1",
					Namespace: "default",
					Labels:    map[string]string{"semaphore-agent/autoscaled": "true"},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"endpoint": []byte("testing.com"),
					"token":    []byte("asdasdasd"),
				},
			},
		}...)

		f, _ := NewAgentTypeFinder(c, "default")
		types, err := f.Find()
		assert.NoError(t, err)
		if assert.Len(t, types, 1) {
			assert.Equal(t, types[0].Name, "agent-type-1")
			assert.Equal(t, types[0].Token, "asdasdasd")
			assert.Equal(t, types[0].Endpoint, "testing.com")
		}
	})
}

func newTestScheme() *runtime.Scheme {
	s := runtime.NewScheme()

	s.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "SecretList",
	}, &corev1.SecretList{})

	s.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Secret",
	}, &corev1.Secret{})

	return s
}

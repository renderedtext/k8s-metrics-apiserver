package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/semaphoreci/k8s-metrics-apiserver/pkg/common"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// We cache the agent type secret information
// to avoid going to the Kubernetes on every iteration.
// But we should also reach to changes to the agent type secrets,
// so we put an expiration on them.
var SecretCacheTTL = 5 * time.Minute

type AgentTypeFinder struct {
	secretsInterface dynamic.ResourceInterface
	cache            *ristretto.Cache
}

func NewAgentTypeFinder(client dynamic.Interface, namespace string) (*AgentTypeFinder, error) {
	// The provider needs read access to secrets,
	// so it can find all the secrets for each agent type,
	// and use the agent type token in them to grab metrics from the Semaphore API.
	secretsInterface := client.
		Resource(schema.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "secrets",
		}).
		Namespace(namespace)

	/*
	 * We keep at most 50 agent types in our cache.
	 */
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 500,
		MaxCost:     50,
		BufferItems: 64,
		Metrics:     false,
	})

	if err != nil {
		return nil, err
	}

	return &AgentTypeFinder{
		secretsInterface: secretsInterface,
		cache:            cache,
	}, nil
}

func (f *AgentTypeFinder) Find() ([]*common.AgentType, error) {
	list, err := f.secretsInterface.List(context.Background(), v1.ListOptions{
		LabelSelector: "semaphore-agent/autoscaled=true",
	})

	if err != nil {
		return []*common.AgentType{}, fmt.Errorf("error listing secrets: %v", err)
	}

	agentTypes := []*common.AgentType{}
	for _, secret := range list.Items {
		agentType, err := f.findAgentType(secret.GetName())
		if err != nil {
			return []*common.AgentType{}, fmt.Errorf("error converting secret '%s' to agent type information: %v", secret.GetName(), err)
		}

		agentTypes = append(agentTypes, agentType)
	}

	return agentTypes, nil
}

// Get the agent type information (endpoint and token) from the secret specified.
// We also cache this information to avoid going to the Kubernetes API on every iteration.
func (f *AgentTypeFinder) findAgentType(secretName string) (*common.AgentType, error) {
	value, found := f.cache.Get(secretName)
	if found {
		if info, ok := value.(*common.AgentType); ok {
			return info, nil
		}
	}

	// If the agent type info does not exist in the cache,
	// we fetch the information from the Kubernetes API.
	o, err := f.secretsInterface.Get(context.Background(), secretName, v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error describing secret: %v", err)
	}

	info, err := f.unstructuredSecretToAgentType(o)
	if err != nil {
		return nil, fmt.Errorf("error finding agent type information in secret: %v", err)
	}

	f.cache.SetWithTTL(secretName, info, 1, SecretCacheTTL)
	return info, nil
}

func (f *AgentTypeFinder) unstructuredSecretToAgentType(secret *unstructured.Unstructured) (*common.AgentType, error) {
	endpoint, err := getNestedString(secret, "data", "endpoint")
	if err != nil {
		return nil, err
	}

	token, err := getNestedString(secret, "data", "token")
	if err != nil {
		return nil, err
	}

	return &common.AgentType{
		Name:     secret.GetName(),
		Endpoint: endpoint,
		Token:    token,
	}, nil
}

func getNestedString(o *unstructured.Unstructured, fields ...string) (string, error) {
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

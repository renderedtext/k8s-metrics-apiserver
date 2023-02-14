## API service for Semaphore metrics

This project exposes an external metrics provider to fetch Semaphore metrics. It's built upon [custom-metrics-apiserver](https://github.com/kubernetes-sigs/custom-metrics-apiserver).

## Running

```bash
# Build provider image
make build
make build.image

# Build agent image
make build.agent.image

# Create semaphore namespace
kubectl apply -f k8s/namespace.yml

# Create resources for custom metrics server
make k8s.metrics.create

# Create resources for agents
make k8s.agent.create
```

## Destroying

```bash
# Delete agent resources
make k8s.agent.delete

# Delete custom metrics server resources
make k8s.metrics.delete
```
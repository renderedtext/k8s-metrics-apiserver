# Custom metrics server for Semaphore

This project exposes an external metrics provider to fetch Semaphore metrics. It's built upon [custom-metrics-apiserver](https://github.com/kubernetes-sigs/custom-metrics-apiserver). The metrics exposed by this server can be used when configuring a Kubernetes [HorizontalPodAutoscaler](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/) to scale a [Semaphore agent](https://github.com/semaphoreci/agent) pool.

Check the [Semaphore agent Helm chart](https://github.com/renderedtext/helm-charts) for usage.

## Metrics exposed

- `agents_total`
- `agents_idle`
- `agents_occupied`
- `agents_occupied_percentage`
- `jobs_total`
- `jobs_running`
- `jobs_queued`

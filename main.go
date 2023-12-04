package main

import (
	"flag"
	"net/http"
	"os"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"

	semaphoreProvider "github.com/semaphoreci/k8s-metrics-apiserver/pkg/provider"
	"github.com/semaphoreci/k8s-metrics-apiserver/pkg/semaphore"
	basecmd "sigs.k8s.io/custom-metrics-apiserver/pkg/cmd"
)

type SemaphoreAdapter struct {
	basecmd.AdapterBase
	Message string
}

func (a *SemaphoreAdapter) makeProviderOrDie() *semaphoreProvider.SemaphoreMetricsProvider {
	client, err := a.DynamicClient()
	if err != nil {
		klog.Fatalf("unable to construct dynamic client: %v", err)
	}

	mapper, err := a.RESTMapper()
	if err != nil {
		klog.Fatalf("unable to construct discovery REST mapper: %v", err)
	}

	provider, err := semaphoreProvider.New(semaphoreProvider.Config{
		Client:          client,
		Mapper:          mapper,
		SemaphoreClient: semaphore.NewClient(http.DefaultClient, false),
	})

	if err != nil {
		klog.Fatalf("unable to construct Semaphore provider: %v", err)
	}

	return provider
}

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	// initialize the flags, with one custom flag for the message
	cmd := &SemaphoreAdapter{}
	cmd.Flags().StringVar(&cmd.Message, "msg", "starting semaphore metrics adapter...", "startup message")

	// make sure you get the klog flags
	logs.AddFlags(cmd.Flags())

	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	if err := cmd.Flags().Parse(os.Args); err != nil {
		klog.Fatalf("unable to parse flags: %v", err)
	}

	provider := cmd.makeProviderOrDie()
	cmd.WithExternalMetrics(provider)
	klog.Infof(cmd.Message)

	go provider.Collect()

	if err := cmd.Run(wait.NeverStop); err != nil {
		klog.Fatalf("unable to run custom metrics adapter: %v", err)
	}
}

package app

import (
	"context"
	"flag"
	"os/signal"
	"syscall"

	"github.com/indexer-benchmark/cmd/indexer/app/options"
	"github.com/indexer-benchmark/pkg/benchmark"
	"github.com/indexer-benchmark/pkg/client"
	"github.com/indexer-benchmark/pkg/mock"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"
)

func NewBenchmarkCommand() *cobra.Command {
	c := options.NewListConfig()
	_ = flag.CommandLine.Parse([]string{})

	cmd := &cobra.Command{
		Use: "benchmark",
		Run: func(cmd *cobra.Command, args []string) {
			_ = flag.Set("v", c.Verbose)
			_ = flag.Set("logtostderr", "true")

			klog.Info(c.String())

			ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
			defer cancel()

			clients := client.CreateKubeClients(client.GetKubeConfig(c.Kubeconfig), c.NumClients)
			qpsLoader := benchmark.NewQpsLoader(c, clients)
			if err := qpsLoader.ListObjects(ctx); err != nil {
				klog.Fatal(err)
			}

			<-ctx.Done()
		},
	}
	c.AddFlags(cmd.Flags())

	return cmd
}

func NewMockCommand() *cobra.Command {
	c := options.NewCreateConfig()
	_ = flag.CommandLine.Parse([]string{})

	cmd := &cobra.Command{
		Use: "mock-data",
		Run: func(cmd *cobra.Command, args []string) {
			_ = flag.Set("v", c.Verbose)
			_ = flag.Set("logtostderr", "true")

			klog.Info(c.String())

			ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
			defer cancel()

			client, err := kubernetes.NewForConfig(client.GetKubeConfig(c.Kubeconfig))
			if err != nil {
				klog.Fatalf("Failed to create kube client: %v", err)
			}
			creator := mock.NewCreator(c, client)
			if err = creator.CreateObjects(ctx); err != nil {
				klog.Fatal(err)
			}

			<-ctx.Done()
		},
	}
	c.AddFlags(cmd.Flags())

	return cmd
}

func Run() error {
	logs.InitLogs()
	defer logs.FlushLogs()

	cmd := &cobra.Command{
		Use: "indexer",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Help()
			return nil
		},
	}
	cmd.AddCommand(NewBenchmarkCommand())
	cmd.AddCommand(NewMockCommand())
	return cmd.Execute()
}

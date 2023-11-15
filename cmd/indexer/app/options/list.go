package options

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
)

type ListConfig struct {
	Kubeconfig    string
	Namespace     string
	NumClients    int
	PageSize      int
	QpsPattern    string
	TotalDuration time.Duration
	WaitDuration  time.Duration
	Verbose       string
}

func (c *ListConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.Kubeconfig, "kubeconfig", filepath.Join(homedir.HomeDir(), ".kube", "config"), "Absolute path to the kubeconfig file")
	fs.StringVar(&c.Namespace, "namespace", "", "Namespace name where the test objects will be listed")
	fs.IntVar(&c.NumClients, "num-clients", 100, "Number of clients to use for spreading the create calls")
	fs.IntVar(&c.PageSize, "page-size", 0, "Number of objects to list in a single page, i.e `limit` param (0 means no pagination)")
	fs.StringVar(&c.QpsPattern, "qps-pattern", "", "QPS to list the objects for each cycle, example: 10:20:30, there will be 3 cycles with 10qps, 20qps and 30qps")
	fs.DurationVar(&c.TotalDuration, "total-duration", 5*time.Minute, "Total duration for which to run this command")
	fs.DurationVar(&c.WaitDuration, "wait-duration", 3*time.Minute, "Wait time after one cycle is done")
	fs.StringVar(&c.Verbose, "v", "5", "verbose level of logs. range 0~9")
}

func NewListConfig() *ListConfig {
	return &ListConfig{}
}

func (c *ListConfig) GetQps() ([]float64, error) {
	if len(c.QpsPattern) == 0 {
		return nil, fmt.Errorf("qps-pattern is mandatory")
	}
	res := strings.Split(c.QpsPattern, ":")
	qps := make([]float64, 0)
	for _, parallel := range res {
		q, err := strconv.ParseFloat(parallel, 64)
		if err != nil {
			klog.Errorf("error converting qps: %v", err)
			return nil, err
		}
		qps = append(qps, q)
	}
	return qps, nil
}

func (c *ListConfig) String() string {
	return fmt.Sprintf("kubeconfig: %v, namespace: %v, num-clients: %v, page-size: %v, qps-pattern: %v, total-duration: %v, wait-duration: %v, verbose: %v",
		c.Kubeconfig, c.Namespace, c.NumClients, c.PageSize, c.QpsPattern, c.TotalDuration, c.WaitDuration, c.Verbose)
}

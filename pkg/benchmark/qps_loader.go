package benchmark

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"sync"
	"sync/atomic"
	"time"

	"github.com/indexer-benchmark/cmd/indexer/app/options"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
)

type QpsLoader struct {
	listConfig *options.ListConfig
	clients    []*kubernetes.Clientset
}

func NewQpsLoader(listConfig *options.ListConfig, clients []*kubernetes.Clientset) *QpsLoader {
	return &QpsLoader{
		listConfig: listConfig,
		clients:    clients,
	}

}

func (q *QpsLoader) ListObjects(ctx context.Context) error {
	qpsSlice, err := q.listConfig.GetQps()
	if err != nil {
		return err
	}
	for i, qps := range qpsSlice {
		q.listObjects(ctx, qps)
		if i != len(qpsSlice)-1 {
			klog.Infof("wait %v to let api server stable", q.listConfig.WaitDuration)
			time.Sleep(q.listConfig.WaitDuration)
		}
	}
	klog.Infof("list objects completed")
	return nil
}

type ListLatency struct {
	latencies []metav1.Duration
	mutex     sync.RWMutex
}

func NewListLatency() *ListLatency {
	return &ListLatency{
		latencies: make([]metav1.Duration, 0),
	}
}

func (l *ListLatency) Add(latency metav1.Duration) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if len(l.latencies) == 0 {
		l.latencies = make([]metav1.Duration, 0)
	}
	l.latencies = append(l.latencies, latency)
}

func (l *ListLatency) GetLatencies() []metav1.Duration {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.latencies
}

func (q *QpsLoader) listObjects(ctx context.Context, qps float64) {
	klog.Infof("Start benchmark with qps: %v", qps)
	start := time.Now()
	ticker := time.NewTicker(time.Duration(1000000000.0/qps) * time.Nanosecond)
	defer ticker.Stop()

	var (
		totalCount  atomic.Uint64
		failedCount atomic.Uint64
		listLatency = NewListLatency() // we don't record failed list latency
	)
	defer func() {
		fc := failedCount.Load()
		tc := totalCount.Load()
		klog.Infof("%d out of %d requests failed, failure rate: %v%%", fc, tc, float64(fc)/float64(tc)*100)
		report(&totalCount, &failedCount, qps, listLatency.GetLatencies())
	}()

	var wg sync.WaitGroup
	for i := 0; time.Since(start) < q.listConfig.TotalDuration; i++ {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			client := q.clients[i%len(q.clients)]
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := q.listOnce(ctx, client, &totalCount, &failedCount, listLatency); err != nil {
					klog.Errorf("Failed to list pods call: %v", err)
				}
			}()
		}
	}

	wg.Wait()
	klog.Infof("Finished listing objects for a duration of %v and qps of %v", q.listConfig.TotalDuration, qps)
}

func (q *QpsLoader) listOnce(ctx context.Context, client *kubernetes.Clientset, totalCount, failedCount *atomic.Uint64, listLatency *ListLatency) error {
	requestCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	totalCount.Add(1)

	start := time.Now()
	rc, err := client.CoreV1().RESTClient().Get().
		Namespace(q.listConfig.Namespace).
		Resource("pods").
		VersionedParams(&metav1.ListOptions{
			Limit:           int64(q.listConfig.PageSize),
			ResourceVersion: "0", // list pods from apiserver cache
		}, scheme.ParameterCodec).
		Stream(requestCtx)
	if rc != nil {
		// Drain response.body to enable TCP connection reuse.
		// Ref: https://github.com/google/go-github/pull/317)
		io.Copy(ioutil.Discard, rc)
		if rc.Close() != nil {
			klog.Errorf("Failed to close the response: %v", err)
		}
	}
	if err != nil {
		failedCount.Add(1)
		return err
	}

	latency := time.Since(start)
	klog.V(7).Infof("List call took: %v", latency)
	listLatency.Add(metav1.Duration{Duration: latency})
	return nil
}

func report(totalCount, failedCount *atomic.Uint64, qps float64, latencies []metav1.Duration) {
	lr := LoaderReport{
		FailedCount: int(failedCount.Load()),
		QPS:         qps,
		TotalCount:  int(totalCount.Load()),
		PercLatency: extractLatencyMetrics(latencies),
		Unit:        "s",
	}
	lr.Report(fmt.Sprintf("[Benchmark] Lists Pods, QPS: %v", qps))
}

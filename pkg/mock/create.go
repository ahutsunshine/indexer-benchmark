package mock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/indexer-benchmark/cmd/indexer/app/options"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type Creator struct {
	createConfig *options.CreateConfig
	client       *kubernetes.Clientset
	mutex        sync.RWMutex
}

func NewCreator(creatConfig *options.CreateConfig, client *kubernetes.Clientset) *Creator {
	return &Creator{
		createConfig: creatConfig,
		client:       client,
	}
}

func (m *Creator) CreateObjects(ctx context.Context) error {
	objects, err := m.createConfig.GetObjects()
	if err != nil {
		return err
	}
	for _, objNum := range objects {
		if err = m.createNamespace(ctx, objNum); err != nil {
			return err
		}
		if err = m.createDefaultSa(ctx, objNum); err != nil {
			return err
		}
		m.createObjects(ctx, objNum)
	}
	klog.Infof("create objects process completed")
	return nil
}

func (m *Creator) createNamespace(ctx context.Context, num int) error {
	requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	ns := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: generateNamespace(num),
		},
	}

	_, err := m.client.CoreV1().Namespaces().Create(requestCtx, ns, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		klog.Errorf("Failed to create object %v in %v namespace: %v", ns.Name, ns.Namespace, err)
		return err
	}
	return nil
}

func (m *Creator) createDefaultSa(ctx context.Context, num int) error {
	requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	namespace := generateNamespace(num)
	sa := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: namespace,
		},
	}

	_, err := m.client.CoreV1().ServiceAccounts(namespace).Create(requestCtx, sa, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		klog.Errorf("Failed to create object %v in %v namespace: %v", sa.Name, sa.Namespace, err)
		return err
	}
	return nil
}

func (m *Creator) createObjects(ctx context.Context, num int) []error {
	namespace := generateNamespace(num)
	groups := splitGroups(m.createConfig.Threads, num)
	var allErr []error
	for groupIndex, groupCnt := range groups {
		var wg sync.WaitGroup
		var errs []error
		klog.Infof("Ready to create %v object in group %v", groupCnt, groupIndex)
		for index := 0; index < groupCnt; index++ {
			wg.Add(1)
			go func(groupIndex, index int) {
				defer wg.Done()
				index = groupIndex*m.createConfig.Threads + index
				if err := createObject(ctx, m.client, namespace, index); err != nil {
					klog.Errorf("Failed to create object in namespace %v: %v", namespace, err)
					errs = m.AppendError(errs, err)
				}
			}(groupIndex, index)
			wg.Wait()
		}
		if len(errs) > 0 {
			allErr = append(allErr, errs...)
		}
		if groupCnt-len(errs) > 0 {
			klog.Infof("Create %v objects in namespace %v of group %v", groupCnt-len(errs), namespace, groupIndex)
		} else {
			klog.Infof("Failed to create %v objects in namespace %v of group %v", len(errs), namespace, groupIndex)
		}
	}
	if num-len(allErr) > 0 {
		klog.Infof("Create %v objects in namespace %v", num-len(allErr), namespace)
	} else {
		klog.Infof("Failed to create %v objects in namespace %v", len(allErr), namespace)
	}
	return allErr
}

func (m *Creator) AppendError(errs []error, err error) []error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if err != nil {
		errs = append(errs, err)
	}
	return errs
}

func createObject(ctx context.Context, client *kubernetes.Clientset, namespace string, podSuffix int) error {
	requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	start := time.Now()
	objectName := fmt.Sprintf("pod-%v", podSuffix)
	// TODO: Implement other object-types below.
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      objectName,
			Namespace: namespace,
			// Below label helps selectively list/delete objects created by indexer-benchmark.
			Labels: map[string]string{
				"app":       "indexer-benchmark",
				"createdBy": "indexer",
				"indexer":   objectName,
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyAlways,
			Containers: []corev1.Container{
				{
					Name:    "ubuntu",
					Image:   "hub.tess.io/tess/ubuntu-20.04:hardened",
					Command: []string{"sh", "-c", "tail -f /dev/null"},
				},
			},
		},
		Status: corev1.PodStatus{},
	}

	_, err := client.CoreV1().Pods(namespace).Create(requestCtx, pod, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		klog.Errorf("Failed to create object %v in %v namespace: %v", pod.Name, pod.Namespace, err)
		return err
	}

	klog.V(7).Infof("Created object %v successfully (took %v)", objectName, time.Since(start))
	return nil
}

func splitGroups(threads, objNum int) []int {
	if objNum <= threads {
		return []int{objNum}
	}
	nums := make([]int, 0)
	groups := objNum / threads
	for i := 0; i < groups; i++ {
		nums = append(nums, threads)
	}
	remaining := objNum % threads
	if remaining != 0 {
		nums = append(nums, remaining)
	}
	return nums
}

func generateNamespace(num int) string {
	return fmt.Sprintf("ns-%v", num)
}

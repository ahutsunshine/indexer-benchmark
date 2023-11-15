package options

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
)

type CreateConfig struct {
	Kubeconfig    string
	Threads       int
	objectType    string
	ObjectPattern string
	Verbose       string
}

func (c *CreateConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.Kubeconfig, "kubeconfig", filepath.Join(homedir.HomeDir(), ".kube", "config"), "Absolute path to the kubeconfig file")
	//TODO support more object types
	fs.StringVar(&c.objectType, "object-type", "pods", "Type of objects to create (supported values are 'pods' currently")
	fs.StringVar(&c.ObjectPattern, "object-pattern", "", "Object to create for different numbers in different namespace(ns- prefix), example: 10:20, there will be 2 cycles to create objects with number of 10 in ns-10 namespace, number of 20 in ns-20")
	fs.IntVar(&c.Threads, "threads", 10, "Number of threads to create objects")
	fs.StringVar(&c.Verbose, "v", "5", "verbose level of logs. range 0~9")
}

func NewCreateConfig() *CreateConfig {
	return &CreateConfig{}
}

func (c *CreateConfig) GetObjects() ([]int, error) {
	if len(c.ObjectPattern) == 0 {
		return nil, fmt.Errorf("object-pattern is mandatory")
	}
	res := strings.Split(c.ObjectPattern, ":")
	objects := make([]int, 0)
	for _, objNum := range res {
		q, err := strconv.ParseInt(objNum, 10, 64)
		if err != nil {
			klog.Errorf("error converting object number [%v]: %v", objNum, err)
			return nil, err
		}
		if q <= 0 {
			return nil, fmt.Errorf("object number [%v] should be greater than 0", q)
		}
		objects = append(objects, int(q))
	}
	return objects, nil
}

func (c *CreateConfig) String() string {
	return fmt.Sprintf("kubeconfig: %v, object-type: %v, object-pattern: %v, threads: %v, verbose: %v",
		c.Kubeconfig, c.objectType, c.objectType, c.Verbose)
}

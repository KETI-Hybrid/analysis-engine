package api

import (
	"analysis-engine/pkg/api/crd"
	"analysis-engine/pkg/api/k8s"

	levelv1 "github.com/KETI-Hybrid/keti-controller/apis/level/v1"
	keticlient "github.com/KETI-Hybrid/keti-controller/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type ClientManager struct {
	KetiClient *keticlient.ClientSet
	KubeClient *kubernetes.Clientset
}

func NewClientManager() *ClientManager {
	var err error
	result := &ClientManager{}

	result.KetiClient, err = crd.NewClient()
	if err != nil {
		klog.Errorln(err)
	}
	
	result.KubeClient, err = k8s.NewClient()
	if err != nil {
		klog.Errorln(err)
	}

	return result
}

type Metric struct {
	CPUUsage      float32
	MemoryUsage   float32
	StorageUsage  float32
	NetworkTXByte float64
	NetworkRXByte float64
}

func (cm *ClientManager) GetMetric(nodeName string) levelv1.NodeMetricSpec {
	nodeMetric, err := cm.KetiClient.LevelV1().NodeMetrics().Get(nodeName, metav1.GetOptions{})
	
	if err != nil {
		klog.Errorln(err)
	}
	return nodeMetric.Spec
}

func (cm *ClientManager) GetPodMetric(podName string) levelv1.PodMetricSpec {
	podMetric, _ := cm.KetiClient.LevelV1().PodMetrics().Get(podName, metav1.GetOptions{})
	// if err != nil {
	// 	klog.Errorln(err)
	// }
	return podMetric.Spec
}

package housekeeping

import (
	"analysis-engine/pkg/watcher/mapper"
	"context"
	"time"

	keticlient "github.com/KETI-Hybrid/keti-controller/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type HouseKeeper struct {
	KetiClient *keticlient.ClientSet
	KubeClient *kubernetes.Clientset
}

func NewHouseKeeper(keticlient *keticlient.ClientSet, kubeclient *kubernetes.Clientset) *HouseKeeper {
	return &HouseKeeper{
		KetiClient: keticlient,
		KubeClient: kubeclient,
	}

}

func (hk *HouseKeeper) NodeKeeping(nodeMap mapper.NodeMetricMapper) mapper.NodeMetricMapper {
	nodes, err := hk.KubeClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	
	if err != nil {
		klog.Errorln(err)
	}
	
	for _, node := range nodes.Items {
		nodeMap[node.Name] = time.Now()
	}
	
	return nodeMap
}

func (hk *HouseKeeper) PodKeeping(podMap mapper.PodMetricMapper) mapper.PodMetricMapper {
	pods, err := hk.KubeClient.CoreV1().Pods(corev1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	
	if err != nil {
		klog.Errorln(err)
	}
	
	for _, pod := range pods.Items {
		if pod.Namespace == "cdi" || pod.Namespace == "keti-controller-system" || pod.Namespace == "keti-system" || pod.Namespace == "kube-flannel" || pod.Namespace == "kube-node-lease" || pod.Namespace == "kube-public" || pod.Namespace == "kube-system" || pod.Namespace == "kubevirt" {
			continue
		} else {
			podMap[pod.Name] = time.Now()
		}
	}
	
	return podMap
}

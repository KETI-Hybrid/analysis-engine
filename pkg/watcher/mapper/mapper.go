package mapper

import (
	"analysis-engine/pkg/api"
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type PodMetricMapper map[string]time.Time
type NodeMetricMapper map[string]time.Time

func NewMapper(cm *api.ClientManager) MetricMapper {
	podPrefix := cm.KubeClient.CoreV1().Pods("keti-system")
	labelMap := make(map[string]string)
	labelMap["name"] = "hpc-metric-collector"

	options := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
func NewPodMapper(cm *api.ClientManager) PodMetricMapper {
	pods, err := cm.KubeClient.CoreV1().Pods(corev1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	
	if err != nil {
		klog.Errorln(err)
	
	podMapper := make(map[string]time.Time)
	
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			podMapper[pod.Name] = time.Now()
		}
	}
	
	return podMapper
}

func NewNodeMapper(cm *api.ClientManager) NodeMetricMapper {
	nodes, err := cm.KubeClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	
	if err != nil {
		klog.Errorln(err)
	}
	
	nodeMapper := make(map[string]time.Time)
	
	for _, node := range nodes.Items {
		nodeMapper[node.Name] = time.Now()
	}
	
	return nodeMapper
}

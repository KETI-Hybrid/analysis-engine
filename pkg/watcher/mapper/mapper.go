package mapper

import (
	"analysis-engine/pkg/api"
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
)

type MetricMapper map[string]string

func NewMapper(cm *api.ClientManager) MetricMapper {
	podPrefix := cm.KubeClient.CoreV1().Pods("keti-system")
	labelMap := make(map[string]string)
	labelMap["name"] = "metric-collector"

	options := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	}
	metricPods, err := podPrefix.List(context.Background(), options)
	if err != nil {
		klog.Errorln(err)
	}
	podIPMap := make(map[string]string)
	for _, pod := range metricPods.Items {
		podIPMap[pod.Spec.NodeName] = pod.Status.PodIP
	}
	return podIPMap
}

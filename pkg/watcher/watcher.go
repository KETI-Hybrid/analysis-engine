package watcher

import (
	"analysis-engine/pkg/api"
	"analysis-engine/pkg/watcher/mapper"
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"
)

type Watcher struct {
	WatchInterface watch.Interface
	NodeIPMapper   mapper.MetricMapper
}

func AttachWatcher(cm *api.ClientManager) *Watcher {
	result := &Watcher{}
	var err error
	podPrefix := cm.KubeClient.CoreV1().Pods("keti-system")
	labelMap := make(map[string]string)
	labelMap["name"] = "metric-collector"

	options := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	}

	result.WatchInterface, err = podPrefix.Watch(context.Background(), options)
	if err != nil {
		klog.Errorln(err)
	}

	result.NodeIPMapper = mapper.NewMapper(cm)

	return result
}

func (w *Watcher) StartWatch() {
	for {
		event := <-w.WatchInterface.ResultChan()
		pod := event.Object.(*v1.Pod)
		w.NodeIPMapper[pod.Spec.NodeName] = pod.Status.PodIP
	}
}

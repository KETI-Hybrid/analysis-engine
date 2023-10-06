package watcher

import (
	"analysis-engine/pkg/api"
	"analysis-engine/pkg/watcher/mapper"
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"
)

type Watcher struct {
	WatchInterface watch.Interface
	NodeIPMapper   mapper.MetricMapper

	DeploymentWatcher watch.Interface
	Deploymentmap     map[string]bool
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

	depPrefix := cm.KubeClient.AppsV1().Deployments(corev1.NamespaceAll)
	result.DeploymentWatcher, err = depPrefix.Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		klog.Errorln(err)
	}
	result.Deploymentmap = make(map[string]bool)

	return result
}

func (w *Watcher) StartWatch() {
	for {
		event := <-w.WatchInterface.ResultChan()
		pod := event.Object.(*corev1.Pod)
		w.NodeIPMapper[pod.Spec.NodeName] = pod.Status.PodIP
	}
}

func (w *Watcher) StartDeploymentWatch() {
	for {
		event := <-w.DeploymentWatcher.ResultChan()
		deployment := event.Object.(*appsv1.Deployment)
		switch event.Type {
		case watch.Added:
			if deployment.Namespace == "keti-system" || deployment.Namespace == "kube-system" || deployment.Namespace == "keti-controller-system" {
				continue
			}
			fmt.Println(`** Deployment restart **`)
			fmt.Printf("Update deployment : %s \n", deployment.Name)
			w.Deploymentmap[deployment.Name] = true
		case watch.Deleted:
			delete(w.Deploymentmap, deployment.Name)
		}

	}
}

package watcher

import (
	"analysis-engine/pkg/api"
	"analysis-engine/pkg/watcher/mapper"
	"context"
	"fmt"

	keticlient "github.com/KETI-Hybrid/keti-controller/client"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"
)

type Watcher struct {
	PodMapper         mapper.PodMetricMapper
	NodeMapper        mapper.NodeMetricMapper
	DeploymentWatcher watch.Interface
	Deploymentmap     map[string]bool
	KetiClient        *keticlient.ClientSet
}

func AttachWatcher(cm *api.ClientManager) *Watcher {
	result := &Watcher{}
	var err error

	depPrefix := cm.KubeClient.AppsV1().Deployments(corev1.NamespaceAll)
	result.DeploymentWatcher, err = depPrefix.Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		klog.Errorln(err)
	}
	result.Deploymentmap = make(map[string]bool)
	result.NodeMapper = make(mapper.NodeMetricMapper)
	result.PodMapper = make(mapper.PodMetricMapper)

	return result
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

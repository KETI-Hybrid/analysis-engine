package crd

import (
	ketiv1 "github.com/KETI-Hybrid/keti-controller/api/v1"
	keticlient "github.com/KETI-Hybrid/keti-controller/client"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

func NewClient() (*keticlient.KetiV1Client, error) {
	err := ketiv1.AddToScheme(scheme.Scheme)
	if err != nil {
		klog.Errorln(err)
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Errorln(err)
	}
	return keticlient.NewForConfig(config)
}

package api

import (
	"analysis-engine/pkg/api/crd"
	pb "analysis-engine/pkg/api/grpc"
	"analysis-engine/pkg/api/k8s"
	"context"
	"time"

	keticlient "github.com/KETI-Hybrid/keti-controller/client"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type ClientManager struct {
	KetiClient *keticlient.KetiV1Client
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
}

func (cm *ClientManager) GetMetric(podIP string) *pb.Response {
	host := podIP + ":50051"
	conn, err := grpc.Dial(host, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		klog.Errorln("did not connect: %v", err)
	}
	defer conn.Close()
	metricClient := pb.NewMetricGRPCClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := metricClient.Get(ctx, &pb.Request{})
	if err != nil {
		klog.Errorln("could not request: %v", err)
	}
	return r
}

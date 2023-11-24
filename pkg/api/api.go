package api

import (
	"analysis-engine/pkg/api/crd"
	"analysis-engine/pkg/api/k8s"
	pb "analysis-engine/pkg/api/metric"
	"context"
	"time"

	keticlient "github.com/KETI-Hybrid/keti-controller/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	CPUUsage      int64
	MemoryUsage   int64
	StorageUsage  int64
	NetworkTXByte int64
	NetworkRXByte int64
}

func (cm *ClientManager) GetMetric(podIP string) (*pb.MultiMetric, error) {
	host := podIP + ":9444"
	conn, err := grpc.Dial(host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		klog.Errorln("did not connect: %v", err)
	}
	defer conn.Close()
	metricClient := pb.NewMetricCollectorClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := metricClient.GetMultiMetric(ctx, &pb.Request{})
	// if err != nil {
	// 	klog.Errorf("could not request: %v \n", err)
	// }
	return r, err
}

package worker

import (
	"analysis-engine/pkg/api"
	"analysis-engine/pkg/api/score"
	"analysis-engine/pkg/watcher"
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type Engine struct {
	score.UnimplementedMetricGRPCServer
	Client        *api.ClientManager
	Watcher       *watcher.Watcher
	NodeScore     map[string]float32
	DeploymentMap map[string]bool
}

func InitEngine() *Engine {
	client := api.NewClientManager()
	wtc := watcher.AttachWatcher(client)

	return &Engine{
		Client:        client,
		Watcher:       wtc,
		NodeScore:     make(map[string]float32),
		DeploymentMap: make(map[string]bool),
	}
}

func (e *Engine) Work() {
	go e.StartGRPCServer()
	go e.Watcher.StartWatch()
	go e.Watcher.StartDeploymentWatch()
	for {
		e.nodeJoinCheck()
		e.nodeStatus()
		e.deploymentStatus()
		e.podStatus()
		time.Sleep(time.Second * 5)
	}
}

func (e *Engine) printNodeScore() {
	fmt.Println("** Node status check **")
	for nodeName, score := range e.NodeScore {
		fmt.Printf("%s : %.2f \n", nodeName, score)
	}
}

func (e *Engine) nodeStatus() {
	for nodeName, podIP := range e.Watcher.NodeIPMapper {
		resp := e.Client.GetMetric(podIP)
		cpuUsage := resp.Message["Host_CPU_Percent"].Metric[0].GetGauge().GetValue()
		memoryUsage := resp.Message["Host_Memory_Percent"].Metric[0].GetGauge().GetValue()
		storageUsage := resp.Message["Host_Storage_Percent"].Metric[0].GetGauge().GetValue()
		score := (0.5 * cpuUsage) + (0.3 * memoryUsage) + (0.2 * storageUsage)
		e.NodeScore[nodeName] = float32(score)
	}
	e.printNodeScore()
}

func (e *Engine) nodeJoinCheck() {
	fmt.Println("** Node join check **")

	hybridNodes, err := e.Client.KetiClient.ResourceV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		klog.Errorln(err)
	}

	fmt.Print("Joined node list : ")
	for i, node := range hybridNodes.Items {
		if _, ok := e.NodeScore[node.Name]; !ok {
			e.NodeScore[node.Name] = 0
		}

		if node.Name == "hcp-master" {
			continue
		}

		fmt.Print(node.Name)

		if i+1 < len(hybridNodes.Items) {
			fmt.Print(", ")
		}
	}

	fmt.Print("\n")
}

func (e *Engine) deploymentStatus() {
	fmt.Println("** Deployment status check **")

	deployments, err := e.Client.KubeClient.AppsV1().Deployments(corev1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		klog.Errorln(err)
	}
	var count int64 = 0
	fmt.Print("Detect deployment : ")
	for _, deployment := range deployments.Items {
		if deployment.Namespace == "keti-system" || deployment.Namespace == "kube-system" || deployment.Namespace == "keti-controller-system" {
			continue
		} else {
			fmt.Print(deployment.Name)
			count += 1
		}
	}
	if count == 0 {
		fmt.Println("None")
	}
}

func (e *Engine) podStatus() {
	for _, podIP := range e.Watcher.NodeIPMapper {
		podMap := e.Client.GetPodMetric(podIP)
		for podName, metric := range podMap {
			if metric.CPUUsage > 60 || metric.MemoryUsage > 60 || metric.StorageUsage > 60 {
				fmt.Println("** Pod Status check **")
				fmt.Println("Pod name :", podName)
				fmt.Println("CPU Usage :", metric.CPUUsage)
				fmt.Println("Memory Usage :", metric.MemoryUsage)
				fmt.Println("Storage Usage :", metric.StorageUsage)
				fmt.Println("NetworkTXByte :", metric.NetworkTXByte)
				fmt.Println("NetworkRXByte :", metric.NetworkRXByte)
			}
		}
	}
}

func (e *Engine) GetNodeScore(ctx context.Context, in *score.Request) (*score.Response, error) {
	res := &score.Response{}
	res.Message = e.NodeScore
	return res, nil
}

func (e *Engine) StartGRPCServer() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		klog.Fatalf("failed to listen: %v", err)
	}
	scoreServer := grpc.NewServer()
	score.RegisterMetricGRPCServer(scoreServer, e)
	fmt.Println("score server started...")
	if err := scoreServer.Serve(lis); err != nil {
		klog.Fatalf("failed to serve: %v", err)
	}
}

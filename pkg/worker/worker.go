package worker

import (
	"analysis-engine/pkg/api"
	"analysis-engine/pkg/api/score"
	"analysis-engine/pkg/watcher"
	"context"
	"fmt"
	"net"
	"strings"
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
		cpuUsage := resp.Message["Host_CPU_Core_Usage"].Metric[0].GetGauge().GetValue()
		memoryUsage := resp.Message["Host_Memory_Usage"].Metric[0].GetGauge().GetValue()
		storageUsage := resp.Message["Host_Storage_Usage"].Metric[0].GetGauge().GetValue()
		networkUsage := resp.Message["Host_Network_Usage"].Metric[0].GetGauge().GetValue()
		score := ((30 * cpuUsage) + (30 * memoryUsage) + (20 * storageUsage) + (networkUsage)) / 81
		e.NodeScore[nodeName] = float32(score)
	}
	e.printNodeScore()
}

func (e *Engine) nodeJoinCheck() {
	logMessage := `** Node join check **
Joined node list :`

	hybridNodes, err := e.Client.KetiClient.ResourceV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		klog.Errorln(err)
	}
	nodeNameList := make([]string, len(hybridNodes.Items))
	for i, node := range hybridNodes.Items {
		if _, ok := e.NodeScore[node.Name]; !ok {
			e.NodeScore[node.Name] = 0
		}
		nodeNameList[i] = node.Name
	}

	nodeNameStr := strings.Join(nodeNameList, ", ")

	fmt.Println(logMessage, nodeNameStr)
}

func (e *Engine) deploymentStatus() {
	fmt.Println("** Deployment status check **")

	deployments, err := e.Client.KubeClient.AppsV1().Deployments(corev1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		klog.Errorln(err)
	}
	for _, deployment := range deployments.Items {
		if deployment.Namespace == "keti-system" || deployment.Namespace == "kube-system" || deployment.Namespace == "keti-controller-system" {
			continue
		}

		fmt.Printf("Detect deployment : %s \n", deployment.Name)
	}
}

func (e *Engine) podStatus() {
	for _, podIP := range e.Watcher.NodeIPMapper {
		podMap := e.Client.GetPodMetric(podIP)
		for podName, metric := range podMap {
			if metric.CPUUsage > 60 || metric.MemoryUsage > 60 {
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

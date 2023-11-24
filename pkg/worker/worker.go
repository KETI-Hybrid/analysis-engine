package worker

import (
	"analysis-engine/pkg/api"
	"analysis-engine/pkg/watcher"
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type Engine struct {
	Client        *api.ClientManager
	Watcher       *watcher.Watcher
	NodeScore     map[string]float32
	DeploymentMap map[string]bool
}

func InitEngine() *Engine {
	fmt.Println("Init worker engine")
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
		if nodeName == "hcp-master" {
			continue
		}

		fmt.Printf("%s : %.2f \n", nodeName, score)
	}
}

func (e *Engine) nodeStatus() {
	for nodeName, podIP := range e.Watcher.NodeIPMapper {
		resp, err := e.Client.GetMetric(podIP)

		if err != nil {
			continue
		}
		node, _ := e.Client.KubeClient.CoreV1().Nodes().Get(context.TODO(), resp.NodeName, metav1.GetOptions{})
		totalCPU, _ := node.Status.Capacity.Cpu().AsInt64()
		totalMemory, _ := node.Status.Capacity.Memory().AsInt64()
		totalStorage := node.Status.Capacity.StorageEphemeral().ToDec().MilliValue()
		cpuUsage := resp.NodeMetric.CpuUsage
		memoryUsage := resp.NodeMetric.MemoryUsage
		storageUsage := resp.NodeMetric.StorageUsage
		CPUPercent := (float32(cpuUsage) * 0.0000001) / float32(totalCPU)
		MemoryPercent := memoryUsage / totalMemory
		StoragePercent := storageUsage / totalStorage
		score := (0.5 * CPUPercent) + (0.3 * float32(MemoryPercent)) + (0.2 * float32(StoragePercent))
		e.NodeScore[nodeName] = float32(score)
	}
	e.printNodeScore()
}

func (e *Engine) nodeJoinCheck() {
	fmt.Println("** Cluster join check **")

	for _, podIP := range e.Watcher.NodeIPMapper {
		resp, err := e.Client.GetMetric(podIP)

		if err != nil {
			continue
		}

		clusterData := resp.NodeMetric.Cluster

		fmt.Print("Joined Cluster : ")
		for i, cluster := range clusterData {
			fmt.Print(cluster)

			if i+1 >= len(clusterData) {
				continue
			} else {
				fmt.Print(",")
			}
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
			fmt.Print(",")
			count += 1
		}
	}

	if count == 0 {
		fmt.Println("None")
	}
	fmt.Print("\n")
}

func (e *Engine) podStatus() {
	for _, podIP := range e.Watcher.NodeIPMapper {
		podMap, err := e.Client.GetMetric(podIP)

		if err != nil {
			continue
		}
		node, _ := e.Client.KubeClient.CoreV1().Nodes().Get(context.TODO(), podMap.NodeName, metav1.GetOptions{})
		totalCPU, _ := node.Status.Capacity.Cpu().AsInt64()
		totalMemory, _ := node.Status.Capacity.Memory().AsInt64()
		totalStorage := node.Status.Capacity.StorageEphemeral().ToDec().MilliValue()
		pod_metric := podMap.NodeMetric.PodMetrics
		for podName, metric := range pod_metric {
			CPUPercent := (float32(metric.CpuUsage) * 0.0000001) / float32(totalCPU)
			MemoryPercent := metric.MemoryUsage / totalMemory
			StoragePercent := metric.StorageUsage / totalStorage
			if CPUPercent > 30 || MemoryPercent > 30 {
				fmt.Println("** Pod Status check **")
				fmt.Println("Pod name :", podName)
				fmt.Println("CPU Usage :", CPUPercent, "%")
				fmt.Println("Memory Usage :", MemoryPercent, "%")
				fmt.Println("Storage Usage :", StoragePercent, "%")
				fmt.Println("NetworkTXByte :", metric.NetworkTx, "Byte")
				fmt.Println("NetworkRXByte :", metric.NetworkRx, "Byte")
			}
		}
	}
}

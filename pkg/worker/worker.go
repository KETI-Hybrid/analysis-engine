package worker

import (
	"analysis-engine/pkg/api"
	"analysis-engine/pkg/api/score"
	"analysis-engine/pkg/housekeeping"
	"analysis-engine/pkg/watcher"
	"analysis-engine/pkg/watcher/mapper"
	"context"
	"fmt"
	"net"
	"strconv"
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
	Keeper        *housekeeping.HouseKeeper
	NodeScore     map[string]float32
	DeploymentMap map[string]bool
}

func InitEngine() *Engine {
	client := api.NewClientManager()
	wtc := watcher.AttachWatcher(client)

	return &Engine{
		Client:        client,
		Watcher:       wtc,
		Keeper:        housekeeping.NewHouseKeeper(client.KetiClient, client.KubeClient),
		NodeScore:     make(map[string]float32),
		DeploymentMap: make(map[string]bool),
	}
}

func (e *Engine) Work() {
	go e.StartGRPCServer()
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
	e.Watcher.NodeMapper = make(mapper.NodeMetricMapper)
	e.Watcher.NodeMapper = e.Keeper.NodeKeeping(e.Watcher.NodeMapper)
	for nodeName, _ := range e.Watcher.NodeMapper {
		// klog.Infoln("Node Name = ", nodeName)
		resp := e.Client.GetMetric(nodeName)
		cpuUsage, _ := strconv.ParseFloat(resp.HostCPUPercent.Value, 64)
		// if err != nil {
		// 	klog.Errorln(err)
		// }
		memoryUsage, _ := strconv.ParseFloat(resp.HostMemoryPercent.Value, 64)
		// if err != nil {
		// 	klog.Errorln(err)
		// }
		storageUsage, _ := strconv.ParseFloat(resp.HostStoragyPercent.Value, 64)
		// if err != nil {
		// 	klog.Errorln(err)
		// }
		fmt.Println("Nodename : ", nodeName)
		fmt.Println("cpuUsage : ", cpuUsage)
		fmt.Println("memoryUsage : ", memoryUsage)
		fmt.Println("storageUsage : ", storageUsage)
		score := (0.5 * cpuUsage) + (0.3 * memoryUsage) + (0.2 * storageUsage)
		e.NodeScore[nodeName] = float32(score)
	}
	e.printNodeScore()
}

func (e *Engine) nodeJoinCheck() {
	fmt.Println("** Node join check **")

	hybridNodes, _ := e.Client.KetiClient.ResourceV1().Nodes().List(metav1.ListOptions{})
	// if err != nil {
	// 	klog.Errorln(err)
	// }

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

	deployments, _ := e.Client.KubeClient.AppsV1().Deployments(corev1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	// if err != nil {
	// 	klog.Errorln(err)
	// }
	var count int64 = 0
	fmt.Print("Detect deployment : ")
	for i, deployment := range deployments.Items {
		if deployment.Namespace == "cdi" || deployment.Namespace == "keti-controller-system" || deployment.Namespace == "keti-system" || deployment.Namespace == "kube-flannel" || deployment.Namespace == "kube-node-lease" || deployment.Namespace == "kube-public" || deployment.Namespace == "kube-system" || deployment.Namespace == "kubevirt" {
			continue
		} else {
			if i < len(deployments.Items)-1 {
				fmt.Print(deployment.Name, ", ")
			}
			count += 1
		}
	}

	fmt.Print("\n")
	if count == 0 {
		fmt.Println("None")
	}
}

func (e *Engine) podStatus() {
	fmt.Println("** Pod Status check **")
	overPod := false
	e.Watcher.PodMapper = make(mapper.PodMetricMapper)
	e.Watcher.PodMapper = e.Keeper.PodKeeping(e.Watcher.PodMapper)
	for podName, _ := range e.Watcher.PodMapper {
		metric := e.Client.GetPodMetric(podName)
		CPUUsage, _ := strconv.ParseFloat(metric.CPUCoreGauge.Value, 64)
		// if err != nil {
		// 	klog.Errorln(err)
		// }
		MemoryUsage, _ := strconv.ParseFloat(metric.MemoryGauge.Value, 64)
		// if err != nil {
		// 	klog.Errorln(err)
		// }
		StorageUsage, _ := strconv.ParseFloat(metric.StorageGauge.Value, 64)
		// if err != nil {
		// 	klog.Errorln(err)
		// }
		NetworkTXByte, _ := strconv.ParseFloat(metric.NetworkTXCounter.Value, 64)
		// if err != nil {
		// 	klog.Errorln(err)
		// }
		NetworkRXByte, _ := strconv.ParseFloat(metric.NetworkRXCounter.Value, 64)
		// if err != nil {
		// 	klog.Errorln(err)
		// }
		if CPUUsage > 50 || MemoryUsage > 50 || StorageUsage > 50 {
			overPod = true
			fmt.Println("Pod name :", podName)
			fmt.Println("CPU Usage :", CPUUsage)
			fmt.Println("Memory Usage :", MemoryUsage)
			fmt.Println("Storage Usage :", StorageUsage)
			if NetworkRXByte < 0 {
				NetworkRXByte = 0
			}
			if NetworkTXByte < 0 {
				NetworkTXByte = 0
			}
			fmt.Println("NetworkTXByte :", NetworkTXByte)
			fmt.Println("NetworkRXByte :", NetworkRXByte)

			pods, _ := e.Client.KubeClient.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
			// if err != nil {
			// 	klog.Errorln(err)
			// }
			for _, pod := range pods.Items {
				if pod.Name == podName {
					if pod.Namespace == "cdi" || pod.Namespace == "keti-controller-system" || pod.Namespace == "keti-system" || pod.Namespace == "kube-flannel" || pod.Namespace == "kube-node-lease" || pod.Namespace == "kube-public" || pod.Namespace == "kube-system" || pod.Namespace == "kubevirt" {
						continue
					} else {
						sec := int64(0)
						err := e.Client.KubeClient.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{GracePeriodSeconds: &sec})
						if err != nil {
							klog.Errorln(err)
						}
					}
				}
			}
		}
	}
	if !overPod {
		fmt.Println("Current Overload Pod : None")
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

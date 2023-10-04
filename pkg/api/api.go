package api

import (
	"analysis-engine/pkg/api/crd"
	pb "analysis-engine/pkg/api/grpc"
	"analysis-engine/pkg/api/k8s"
	"context"
	"fmt"
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
	CPUUsage      float32
	MemoryUsage   float32
	StorageUsage  float32
	NetworkTXByte float64
	NetworkRXByte float64
}

func (cm *ClientManager) GetMetric(podIP string) *pb.Response {
	host := podIP + ":50051"
	conn, err := grpc.Dial(host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		klog.Errorln("did not connect: %v", err)
	}
	defer conn.Close()
	metricClient := pb.NewMetricGRPCClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := metricClient.Get(ctx, &pb.Request{})
	if err != nil {
		klog.Errorf("could not request: %v \n", err)
	}
	return r
}

func (cm *ClientManager) GetPodMetric(podIP string) map[string]Metric {
	host := podIP + ":50051"
	conn, err := grpc.Dial(host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		klog.Errorln("did not connect: %v", err)
	}
	defer conn.Close()
	metricClient := pb.NewMetricGRPCClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := metricClient.Get(ctx, &pb.Request{})
	if err != nil {
		klog.Errorf("could not request: %v \n", err)
	}
	response := make(map[string]Metric)
	for _, cpuVal := range r.Message["CPU_Core_Gauge"].Metric {
		if len(cpuVal.GetLabel()) == 4 {
			namespacedName := fmt.Sprintf("%s/%s", cpuVal.GetLabel()[2].GetValue(), cpuVal.GetLabel()[2].GetValue())
			if _, ok := response[namespacedName]; ok {
				response[namespacedName] = Metric{
					CPUUsage:      float32(cpuVal.GetGauge().GetValue()),
					MemoryUsage:   response[namespacedName].MemoryUsage,
					StorageUsage:  response[namespacedName].StorageUsage,
					NetworkTXByte: response[namespacedName].NetworkTXByte,
					NetworkRXByte: response[namespacedName].NetworkRXByte,
				}
			} else {
				response[namespacedName] = Metric{
					CPUUsage:      float32(cpuVal.GetGauge().GetValue()),
					MemoryUsage:   0,
					StorageUsage:  0,
					NetworkTXByte: 0,
					NetworkRXByte: 0,
				}
			}
		}
	}
	for _, memoryVal := range r.Message["Memory_Gauge"].Metric {
		if len(memoryVal.GetLabel()) == 4 {
			namespacedName := fmt.Sprintf("%s/%s", memoryVal.GetLabel()[2].GetValue(), memoryVal.GetLabel()[2].GetValue())
			if _, ok := response[namespacedName]; ok {
				response[namespacedName] = Metric{
					CPUUsage:      response[namespacedName].CPUUsage,
					MemoryUsage:   float32(memoryVal.GetGauge().GetValue()),
					StorageUsage:  response[namespacedName].StorageUsage,
					NetworkTXByte: response[namespacedName].NetworkTXByte,
					NetworkRXByte: response[namespacedName].NetworkRXByte,
				}
			} else {
				response[namespacedName] = Metric{
					CPUUsage:      0,
					MemoryUsage:   float32(memoryVal.GetGauge().GetValue()),
					StorageUsage:  0,
					NetworkTXByte: 0,
					NetworkRXByte: 0,
				}
			}
		}
	}
	for _, storageVal := range r.Message["Storage_Gauge"].Metric {
		if len(storageVal.GetLabel()) == 4 {
			namespacedName := fmt.Sprintf("%s/%s", storageVal.GetLabel()[2].GetValue(), storageVal.GetLabel()[2].GetValue())
			if _, ok := response[namespacedName]; ok {
				response[namespacedName] = Metric{
					CPUUsage:      response[namespacedName].CPUUsage,
					MemoryUsage:   response[namespacedName].MemoryUsage,
					StorageUsage:  float32(storageVal.GetGauge().GetValue()),
					NetworkTXByte: response[namespacedName].NetworkTXByte,
					NetworkRXByte: response[namespacedName].NetworkRXByte,
				}
			} else {
				response[namespacedName] = Metric{
					CPUUsage:      0,
					MemoryUsage:   0,
					StorageUsage:  float32(storageVal.GetGauge().GetValue()),
					NetworkTXByte: 0,
					NetworkRXByte: 0,
				}
			}
		}
	}
	for _, txVal := range r.Message["Network_TX_Counter"].Metric {
		if len(txVal.GetLabel()) == 4 {
			namespacedName := fmt.Sprintf("%s/%s", txVal.GetLabel()[2].GetValue(), txVal.GetLabel()[2].GetValue())
			if _, ok := response[namespacedName]; ok {
				currentnetwork := txVal.GetGauge().GetValue() - response[namespacedName].NetworkTXByte
				if currentnetwork > 0 {
					response[namespacedName] = Metric{
						CPUUsage:      response[namespacedName].CPUUsage,
						MemoryUsage:   response[namespacedName].MemoryUsage,
						StorageUsage:  response[namespacedName].StorageUsage,
						NetworkTXByte: currentnetwork,
						NetworkRXByte: response[namespacedName].NetworkRXByte,
					}
				} else {
					response[namespacedName] = Metric{
						CPUUsage:      response[namespacedName].CPUUsage,
						MemoryUsage:   response[namespacedName].MemoryUsage,
						StorageUsage:  response[namespacedName].StorageUsage,
						NetworkTXByte: txVal.GetGauge().GetValue(),
						NetworkRXByte: response[namespacedName].NetworkRXByte,
					}
				}
			} else {
				response[namespacedName] = Metric{
					CPUUsage:      0,
					MemoryUsage:   0,
					StorageUsage:  0,
					NetworkTXByte: txVal.GetGauge().GetValue(),
					NetworkRXByte: 0,
				}
			}
		}
	}
	for _, rxVal := range r.Message["Network_RX_Counter"].Metric {
		if len(rxVal.GetLabel()) == 4 {
			namespacedName := fmt.Sprintf("%s/%s", rxVal.GetLabel()[2].GetValue(), rxVal.GetLabel()[2].GetValue())
			if _, ok := response[namespacedName]; ok {
				currentnetwork := rxVal.GetGauge().GetValue() - response[namespacedName].NetworkRXByte
				if currentnetwork > 0 {
					response[namespacedName] = Metric{
						CPUUsage:      response[namespacedName].CPUUsage,
						MemoryUsage:   response[namespacedName].MemoryUsage,
						StorageUsage:  response[namespacedName].StorageUsage,
						NetworkTXByte: response[namespacedName].NetworkTXByte,
						NetworkRXByte: currentnetwork,
					}
				} else {
					response[namespacedName] = Metric{
						CPUUsage:      response[namespacedName].CPUUsage,
						MemoryUsage:   response[namespacedName].MemoryUsage,
						StorageUsage:  response[namespacedName].StorageUsage,
						NetworkTXByte: response[namespacedName].NetworkTXByte,
						NetworkRXByte: rxVal.GetGauge().GetValue(),
					}
				}
			} else {
				response[namespacedName] = Metric{
					CPUUsage:      0,
					MemoryUsage:   0,
					StorageUsage:  0,
					NetworkTXByte: 0,
					NetworkRXByte: rxVal.GetGauge().GetValue(),
				}
			}
		}
	}
	return response
}

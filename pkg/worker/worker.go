package worker

import (
	"analysis-engine/pkg/api"
	"analysis-engine/pkg/watcher"
	"time"

	"k8s.io/klog/v2"
)

type Engine struct {
	Client    *api.ClientManager
	Watcher   *watcher.Watcher
	NodeScore map[string]float32
}

func InitEngine() *Engine {
	client := api.NewClientManager()
	wtc := watcher.AttachWatcher(client)

	return &Engine{
		Client:    client,
		Watcher:   wtc,
		NodeScore: make(map[string]float32),
	}
}

func (e *Engine) Work() {
	go e.Watcher.StartWatch()
	for {
		for nodeName, podIP := range e.Watcher.NodeIPMapper {
			resp := e.Client.GetMetric(podIP)
			cpuUsage := resp.Message["Host_CPU_Core_Usage"].Metric[0].GetGauge().GetValue()
			memoryUsage := resp.Message["Host_Memory_Usage"].Metric[0].GetGauge().GetValue()
			storageUsage := resp.Message["Host_Storage_Usage"].Metric[0].GetGauge().GetValue()
			score := ((3 * cpuUsage) + (2 * memoryUsage) + (storageUsage)) / 3
			e.NodeScore[nodeName] = float32(score)
			e.setLevels(float32(score))
		}

		time.Sleep(time.Second * 5)
	}

}

func (e *Engine) setLevels(score float32) {
	klog.Infoln("Current Score : ", score)
	klog.Infoln("Current Warning Level : 1")
	klog.Infoln("Current Watching Level : 1")
	klog.Infoln("Current Rebalancing Level : 1")
}

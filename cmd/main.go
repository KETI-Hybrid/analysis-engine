package main

import "analysis-engine/pkg/worker"

func main() {
	w := worker.InitEngine()
	w.Work()
}

package main

import (
	"analysis-engine/pkg/worker"
	"fmt"
)

func main() {
	fmt.Println("Start analysis engine")
	w := worker.InitEngine()
	w.Work()
}

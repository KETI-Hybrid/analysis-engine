#!/bin/bash
controller_name="hybrid.analysis-engine"
export GO111MODULE=on
go mod vendor
go build -o /root/workspace/hth/dev/analysis-engine/build/_output/bin/$controller_name -mod=vendor /root/workspace/hth/dev/analysis-engine/cmd/main.go


#0.0.2 => 운용중인 metric collector member
#0.0.3 => test 용 버전 (제병)
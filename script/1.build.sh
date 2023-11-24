controller_name="hybrid.analysis-engine"
docker_id="ketidevit2"
export GO111MODULE=on
go mod vendor
go build -o /root/workspace/pjh/dev/analysis-engine/build/_output/bin/$controller_name -mod=vendor /root/workspace/pjh/dev/analysis-engine/cmd/main.go
=======
go build -o /root/workspace/hth/dev/analysis-engine/build/_output/bin/$controller_name -mod=vendor /root/workspace/hth/dev/analysis-engine/cmd/main.go
docker build -t $docker_id/$controller_name:latest /root/workspace/hth/dev/analysis-engine/build && \
docker push $docker_id/$controller_name:latest

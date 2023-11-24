#!/bin/bash
docker_id="ketidevit2"
controller_name="hybrid.analysis-engine"
docker build -t $docker_id/$controller_name:latest /root/workspace/pjh/dev/analysis-engine/build && \
docker push $docker_id/$controller_name:latest

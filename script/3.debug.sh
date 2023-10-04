#/bin/bash
NS=keti-system
CLUSTER=cluster1
NAME=$(kubectl get pod -n $NS | grep -E 'analysis-engine' | awk '{print $1}')

#echo "Exec Into '"$NAME"'"

#kubectl exec -it $NAME -n $NS /bin/sh
kubectl logs -f -n $NS $NAME


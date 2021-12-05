## K8s Cluster Setup 
#### kops Kubernetes Cluster Setup on Amazon Web Services according to https://kops.sigs.k8s.io/getting_started/aws/ #####

##### Install prerequisites for local machine
```
curl -LO https://github.com/kubernetes/kops/releases/download/v1.21.0/kops-darwin-amd64
chmod +x kops-darwin-amd64
sudo mv kops-darwin-amd64 /usr/local/bin/kops
brew install kubernetes-cli
pip install awscli
```
Note: Latest kOps version (1.22) breaks the extender

#####  Set up AWS according to https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/setting-up.html

#####  install Go according to https://golang.org/doc/install

#####  verify Go installation
```
go version 
```

#####  install aws SDK for Go 
```
go get -u github.com/aws/aws-sdk-go/...
```

#####  aws cli config
```
nano .aws/credentials
export AWS_REGION=us-east-1
```

#####  create kops user
```
aws iam create-group --group-name kops

aws iam attach-group-policy --policy-arn arn:aws:iam::aws:policy/AmazonEC2FullAccess --group-name kops
aws iam attach-group-policy --policy-arn arn:aws:iam::aws:policy/AmazonRoute53FullAccess --group-name kops
aws iam attach-group-policy --policy-arn arn:aws:iam::aws:policy/AmazonS3FullAccess --group-name kops
aws iam attach-group-policy --policy-arn arn:aws:iam::aws:policy/IAMFullAccess --group-name kops
aws iam attach-group-policy --policy-arn arn:aws:iam::aws:policy/AmazonVPCFullAccess --group-name kops

aws iam create-user --user-name kops

aws iam add-user-to-group --user-name kops --group-name kops

aws iam create-access-key --user-name kops
```


#####  configure the aws client to use the kops user
```
aws configure           
aws iam list-users

export AWS_ACCESS_KEY_ID=$(aws configure get aws_access_key_id)
export AWS_SECRET_ACCESS_KEY=$(aws configure get aws_secret_access_key)
```

#####  create S3 bucket to store cluster state
``` 
aws s3api create-bucket \
    --bucket greenk8s-ha-state-store \
    --region us-east-1

aws s3api put-bucket-versioning --bucket greenk8s-ha-state-store  --versioning-configuration Status=Enabled
aws s3api put-bucket-encryption --bucket greenk8s-ha-state-store --server-side-encryption-configuration '{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"AES256"}}]}'
```

#####  define availability zones
```
aws ec2 describe-availability-zones --region us-east-1
```

#####  create key pair
```
aws ec2 create-key-pair --key-name greenkey --query "KeyMaterial" --output text > greenkey.pem
chmod 400 greenkey.pem
```

#####  set env variables for the cluster (once the previous steps are done, restart from here)
```
export NAME=ha.greenk8s.com
export KOPS_STATE_STORE=s3://greenk8s-ha-state-store
```

#####  create cluster
```
kops create cluster \
    --node-count=3 \
    --master-count=1 \
    --master-size=t2.medium \
    --node-size=t2.small \
    --zones="us-east-1a" \
    --cloud-labels="purpose=thesis" \
    ${NAME}
```

##### start cluster
```
kops update cluster --name ${NAME} --yes --admin
```

#####  check if everything is working
```
kops validate cluster 
kubectl get nodes
```

#####  test ssh into master node (in Case connection fails: https://www.thegeekdiary.com/how-to-fix-the-error-host-key-verification-failed/)
```
ssh -i ~/.ssh/id_rsa ubuntu@api.ha.greenk8s.com
```

#####  check nodes and pods
``` 
kubectl get nodes -o wide
```

#####  delete cluster
```
kops delete cluster --name ${NAME} --yes
```

#####  Troubleshooting 
https://www.poeticoding.com/create-a-high-availability-kubernetes-cluster-on-aws-with-kops/
https://serverfault.com/questions/993888/kubernetes-with-kops-in-aws-how-to-attach-iam-policies-to-the-iam-role-used-to

For kubectl error: You must be logged in to the server (Unauthorized) set state store env variable and export kubecfg:
```
kops export kubecfg --admin 
kubectl get --raw /apis/metrics.k8s.io/v1beta1/nodes
```

## Deploy Scheduler Extension

#####  Build Docker Image:
```
export IMAGE=timokraus/green-k8s-scheduler:latest
docker build . -t "${IMAGE}"
docker push "${IMAGE}"
```

#####  Run extender image
```
kubectl apply -f scheduler.yaml
```

#####  Check if pod was created:
```
kubectl get pods --all-namespaces 
kubectl get pods -n kube-system
```

#####  Stream Logs for troubleshooting:
```
kubectl -n kube-system logs deploy/green-k8s-scheduler -c green-k8s-scheduler-extender-ctr -f
```

#####  Deploy test pods:
```
kubectl apply -f Deployment.yaml
```

#####  Troubleshooting: 
```
kubectl describe pod nginx-deployment-7bcbdc8dfd-24224

kubectl logs green-k8s-descheduler-56d745498f-km96r -n kube-system -p
kubectl describe pods green-k8s-scheduler-584f5649b8-4dgxv  -n kube-system
kubectl logs -f green-k8s-scheduler-584f5649b8-4dgxv -c green-k8s-scheduler-extender-ctr -p
kubectl -n kube-system logs deploy/green-k8s-scheduler -c green-k8s-scheduler -f
kubectl get deployment green-k8s-descheduler --namespace=kube-system
kubectl delete deployment green-k8s-descheduler -n kube-system
```
# Manual annotation / labeling
```
kubectl label nodes ip-172-20-90-143.ec2.internal green=true
kubectl get nodes --show-labels
kubectl annotate nodes <your-node-name> renewable=0.6
```

# Manual tainting

kubectl taint nodes ip-172-20-88-199.ec2.internal green=false:NoSchedule-
kubectl taint nodes ip-172-20-88-199.ec2.internal green=false:NoExecute-

Liveliness Probe?

# Samples
https://developer.ibm.com/articles/creating-a-custom-kube-scheduler/
https://github.com/everpeace/k8s-scheduler-extender-example



## Install Prometheus/Grafana Monitoring
https://computingforgeeks.com/setup-prometheus-and-grafana-on-kubernetes/

##### forward prometheus dashboard port http://localhost:9090
```
kubectl port-forward -n monitoring prometheus-green-k8s-monitor-kube-pro-prometheus-0 9090
```
##### forward grafana dashboard port http://localhost:3000
```
kubectl port-forward green-k8s-monitor-grafana-6dc6596dff-xmhcl 3000 -n monitoring
```
configure using localhost:9090 and "browser"


https://serverfault.com/questions/1042202/how-can-i-measure-pod-startup-time-in-kubernetes
https://www.youtube.com/watch?v=q8MFm2jwXpA

## Run the Electricity Simulator

##### Troubleshooting
https://stackoverflow.com/questions/59741353/cannot-patch-kubernetes-node-using-python-kubernetes-client-library
https://deepdive.tw/2017/01/04/installing-kubernetes-on-aws-with-kops/


## Taint nodes
kubectl taint

http://doc.forecast.solar/doku.php?id=api:estimate

## Create Dashboard

https://github.com/kubernetes/dashboard


### aob
https://stackoverflow.com/questions/62803041/how-to-evict-or-delete-pods-from-kubernetes-using-golang-client
https://stackoverflow.com/questions/53857593/how-to-get-status-of-a-pod-in-kubernetes-using-go-client
https://math.stackexchange.com/questions/684519/what-is-the-most-scientific-way-to-assign-weights-to-historical-data/684629

### metrics 
https://stackoverflow.com/questions/52763291/get-current-resource-usage-of-a-pod-in-kubernetes-with-go-client
https://stackoverflow.com/questions/52029656/how-to-retrieve-kubernetes-metrics-via-client-go-and-golang
https://www.datadoghq.com/blog/how-to-collect-and-graph-kubernetes-metrics/
https://thecloudblog.net/lab/practical-top-down-resource-monitoring-of-a-kubernetes-cluster-with-metrics-server/
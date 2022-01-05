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
    --node-count=4 \
    --master-count=1 \
    --master-size=t2.medium \
    --node-size=t2.small \
    --zones=us-east-1a,us-east-1b,us-east-1c,us-east-1d \
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
kube-scheduler config: /etc/kubernetes/manifests
kube-scheduler logs: /var/log/kube-scheduler.log
sudo grep -i 'score' kube-scheduler.log
```

#####  check nodes and pods
``` 
kubectl get nodes -o wide
```

#####  run metrics-server (needed to get CPU utilization)
```
kubectl apply -f manifests/metrics-server.yaml
kubectl get --raw /apis/metrics.k8s.io/v1beta1/nodes
kubectl top nodes
```

#####  delete cluster
```
kops delete cluster --name ${NAME} --yes
```

#####  Troubleshooting 
For kubectl error: You must be logged in to the server (Unauthorized) set state store env variable and export kubecfg:
```
kops export kubecfg --admin 
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
kubectl apply -f manifests/scheduler.yaml
```

#####  Check if pod was created:
```
kubectl get pods --all-namespaces 
kubectl get pods -n kube-system
```

#####  Stream Logs for troubleshooting:
```
kubectl -n kube-system logs deploy/green-k8s-scheduler -c green-k8s-scheduler-extender-ctr -f

kubectl -n kube-system logs deploy/green-k8s-scheduler -c green-k8s-scheduler-ctr -f > scheduler.log
```

#####  Deploy test pods:
```
kubectl apply -f manifests/deployment.yaml
```
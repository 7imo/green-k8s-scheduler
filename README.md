## K8s Cluster Setup 
#### kops Kubernetes Cluster Setup on Amazon Web Services according to https://kops.sigs.k8s.io/getting_started/aws/ #####

##### Install prerequisites for local machine
```
brew update && brew install kops
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
    --node-count=2 \
    --master-count=1 \
    --master-size=t2.large \
    --node-size=t2.medium \
    --zones="us-east-1a,us-east-1c" \
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


## Deploy Scheduler Extension

#####  Build Docker Image:
```
export IMAGE=timokraus/green-k8s-scheduler:latest
docker build . -t "${IMAGE}"
docker push "${IMAGE}"
```

#####  Run extender image
```
sed 's/a\/b:c/'$(echo "${IMAGE}" | sed 's/\//\\\//')'/' extender.yaml | kubectl apply -f -
```

#####  Show if Pod was created:
```
kubectl get pods --all-namespaces 
kubectl get pods -n kube-system
```

#####  Stream Logs for troubleshooting:
```
kubectl -n kube-system logs deploy/green-k8s-scheduler -c green-k8s-scheduler-extender-ctr -f
```

#####  Deploy Config:
```
kubectl apply -f Deployment.yaml
```

#####  Verify:
```
kubectl describe pod nginx-deployment-7bcbdc8dfd-24224
```

#####  Troubleshooting: 
```
kubectl logs green-k8s-scheduler-6789dd9f7-fjpnp -n kube-system -p
kubectl describe pods green-k8s-scheduler-6789dd9f7-fjpnp -n kube-system
kubectl logs -f green-k8s-scheduler-6789dd9f7-fjpnp -c green-k8s-scheduler-extender-ctr -p

kubectl label nodes <your-node-name> green=true
kubectl annotate nodes <your-node-name> renewable=0.6
```

Liveliness Probe?
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

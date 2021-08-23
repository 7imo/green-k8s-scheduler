
### Spin up K8s Cluster ###

### Deploy Scheduler ###

1. Build Docker Image:
export IMAGE=timokraus/green-k8s-scheduler:latest
docker build . -t "${IMAGE}"
docker push "${IMAGE}"

2. Run extender image
sed 's/a\/b:c/'$(echo "${IMAGE}" | sed 's/\//\\\//')'/' extender.yaml | kubectl apply -f -

3. Show if Pod was created:
kubectl get pods --all-namespaces 

4. Stream Logs:
kubectl -n kube-system logs deploy/green-k8s-scheduler -c green-k8s-scheduler-extender-ctr -f

5. Deploy Deployment:
kubectl apply -f Deployment.yaml

6. Verify:
kubectl describe pod nginx-deployment-7bcbdc8dfd-24224


# Troubleshooting: #
kubectl logs green-k8s-scheduler-6789dd9f7-fjpnp -n kube-system -p
kubectl describe pods green-k8s-scheduler-6789dd9f7-fjpnp -n kube-system
kubectl logs -f green-k8s-scheduler-6789dd9f7-fjpnp -c green-k8s-scheduler-extender-ctr -p   
Liveliness Probe?
https://developer.ibm.com/articles/creating-a-custom-kube-scheduler/
https://github.com/everpeace/k8s-scheduler-extender-example



### Install Prometheus ###
brew install helm

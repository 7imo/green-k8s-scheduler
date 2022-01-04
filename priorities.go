package main

import (
	"log"

	extender "k8s.io/kube-scheduler/extender/v1"
)

// scores returned by this function will be added back to default scheduler
func prioritize(args extender.ExtenderArgs) *extender.HostPriorityList {
	pod := args.Pod
	nodes := args.Nodes.Items

	log.Printf("Calculating node priorities for pod %v ...", pod.Name)

	// call green-k8s-scheduler logic
	nodeScoreMap := calculateScoresFromRenewables(args.Nodes)
	hostPriorityList := make(extender.HostPriorityList, len(nodes))

	for i, node := range nodes {
		i64score := int64(nodeScoreMap[node.Name])
		hostPriorityList[i] = extender.HostPriority{
			Host:  node.Name,
			Score: i64score,
		}
	}

	return &hostPriorityList
}

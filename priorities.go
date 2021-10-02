package main

import (
	"log"

	extender "k8s.io/kube-scheduler/extender/v1"
)

// you can't see existing scores calculated so far by default scheduler
// scores output by this function will be added back to default scheduler
func prioritize(args extender.ExtenderArgs) *extender.HostPriorityList {
	pod := args.Pod
	nodes := args.Nodes.Items

	log.Printf("Calculating node priorities for pod %v ...", pod.Name)

	nodeScoreMap := CalculateScoresFromRenewables(args)
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

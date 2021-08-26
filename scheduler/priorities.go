package main

import (
	"log"

	extender "k8s.io/kube-scheduler/extender/v1"
)

// It'd better to only define one custom priority per extender
// as current extender interface only supports one single weight mapped to one extender
// and also it returns HostPriorityList, rather than []HostPriorityList

// it's webhooked to pkg/scheduler/core/generic_scheduler.go#prioritizeNodes()
// you can't see existing scores calculated so far by default scheduler
// instead, scores output by this function will be added back to default scheduler
func prioritize(args extender.ExtenderArgs) *extender.HostPriorityList {
	pod := args.Pod
	nodes := args.Nodes.Items

	log.Printf("Calculating node priorities for pod %v ...", pod.Name)

	nodeScoreMap := calculateScores(args)
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

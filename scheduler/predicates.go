package main

import (
	"log"
	"strings"

	v1 "k8s.io/api/core/v1"
	extender "k8s.io/kube-scheduler/extender/v1"
)

// filter filters nodes according to predicates defined in this extender
func filter(args extender.ExtenderArgs) *extender.ExtenderFilterResult {
	var filteredNodes []v1.Node
	var failReasons []string
	failedNodes := make(extender.FailedNodesMap)
	pod := args.Pod

	log.Printf("Checking node predicates for pod %v ...", pod.Name)

	for _, node := range args.Nodes.Items {
		if node.Labels["green"] == "true" {
			filteredNodes = append(filteredNodes, node)
			log.Printf("Node %v has a green Label. Ready to schedule Pod.", node.Name)
		} else {
			failedNodes[node.Name] = strings.Join(failReasons, "NodeLabelMatchFailure")
			log.Printf("Node %v does not have a green Label. Cannot schedule Pod.", node.Name)
		}
	}

	result := extender.ExtenderFilterResult{
		Nodes: &v1.NodeList{
			Items: filteredNodes,
		},
		FailedNodes: failedNodes,
		Error:       "",
	}

	return &result
}

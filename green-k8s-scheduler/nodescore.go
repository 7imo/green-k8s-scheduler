package main

import (
	"log"

	extender "k8s.io/kube-scheduler/extender/v1"
)

func updateScores(args extender.ExtenderArgs) map[string]int {

	//Make Global
	nodeScoreMap := make(map[string]int)
	nodes := args.Nodes.Items

	//TODO: Match with external data and calculate Score for each Node

	for i, node := range nodes {
		log.Printf("Node %v gets Score %v", node.Name, i)
		nodeScoreMap[node.Name] = 1 + i
	}

	return nodeScoreMap
}

/*
func getScore(node string) int64 {
	log.Printf("Calculating score for Node %v", node)
	return rand.Int63n(5)
}
*/

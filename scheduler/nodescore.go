package main

import (
	"log"
	"strconv"

	extender "k8s.io/kube-scheduler/extender/v1"
)

const MAX_SCORE = 10.0

func calculateScores(args extender.ExtenderArgs) map[string]int {

	nodeScoreMap := make(map[string]int)
	nodes := args.Nodes.Items
	var score int

	// read renewable shares from node annotations
	for i, node := range nodes {
		log.Printf("Node %v is at loop %v", node.Name, i)
		//TODO: Error handling if annotation non exist
		renewableShare, err := strconv.ParseFloat(node.Annotations["renewable"], 32)
		if err != nil {
			log.Printf("error running program: %s \n", err.Error())
		}
		score = int(renewableShare * MAX_SCORE)
		log.Printf("Node %v will be assigned a Score of %v", node.Name, score)
		nodeScoreMap[node.Name] = score
	}

	return nodeScoreMap
}

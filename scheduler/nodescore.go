package main

import (
	"log"
	"strconv"

	extender "k8s.io/kube-scheduler/extender/v1"
)

const MaxScore = 10.0

func calculateScores(args extender.ExtenderArgs) map[string]int {

	nodeScoreMap := make(map[string]int)
	nodes := args.Nodes.Items
	var score int

	//read renewable shares from node annotations
	for i, node := range nodes {
		log.Printf("Node %v is at loop %v", node.Name, i)
		renewableShare, err := strconv.ParseFloat(node.Annotations["renewable"], 32)
		if err != nil {
			log.Printf("error running program: %s \n", err.Error())
		}
		score = int(renewableShare * MaxScore)
		log.Printf("Node %v will be assigned a Score of %v", node.Name, score)
		nodeScoreMap[node.Name] = score
	}

	return nodeScoreMap
}

/*
func getNodeAnnotations(node string) int64 {
	log.Printf("Calculating score for Node %v", node)
	return rand.Int63n(5)
}
*/

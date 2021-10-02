package main

import (
	"log"
	"math"
	"strconv"

	extender "k8s.io/kube-scheduler/extender/v1"
)

const MAX_SCORE = 100

func CalculateScoresFromRenewables(args extender.ExtenderArgs) map[string]int {

	renewables := make(map[string]float64)
	nodes := args.Nodes.Items

	// read renewable shares from node annotations
	for _, node := range nodes {

		log.Printf("Parsing renewable share of node %v", node.Name)
		renewableShare, err := strconv.ParseFloat(node.Annotations["renewable"], 64)
		if err != nil {
			log.Printf("Error parsing renewable share from node: %s \n", err.Error())
			renewableShare = 0
		}

		log.Printf("Node %v has a renewable share of %v", node.Name, renewableShare)
		renewables[node.Name] = float64(renewableShare)
	}

	return NormalizeScores(renewables)
}

func NormalizeScores(renewables map[string]float64) map[string]int {
	highest := 1.0
	scores := make(map[string]int)
	var score int

	for _, renewableShare := range renewables {
		highest = math.Max(highest, renewableShare)
		log.Printf("Highest share so far: %v", highest)
	}

	for node, renewableShare := range renewables {
		score = int(renewableShare * MAX_SCORE / highest)
		scores[node] = score
		log.Printf("Node %v has a score of %v", node, score)
	}

	return scores
}

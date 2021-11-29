package main

import (
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
)

const MAX_SCORE = 10.0

// reads envs from deployment
var mode = os.Getenv("MODE")
var weight = os.Getenv("WEIGHT")

func parseRenewablesFromNodes(nodeList *v1.NodeList) map[string][]float64 {

	renewables := make(map[string][]float64)

	// read renewable shares from node annotations
	for _, node := range nodeList.Items {

		shares_string := node.Annotations["renewable"]
		if shares_string == "" {
			log.Printf("Error parsing renewable share from node %v: No values found. Assigning a renewable energy share of 0.", node.Name)
			shares_string = "0.0;0.0;0.0;0.0;0.0"
		}

		// split string into slice with single values
		shares := strings.Split(shares_string, ";")
		var shares_float []float64

		// convert strings to floats
		for i := 0; i < len(shares); i += 1 {
			f64, _ := strconv.ParseFloat(shares[i], 64)
			shares_float = append(shares_float, float64(f64))
		}

		// Logs for Debugging
		log.Printf("Renewable shares parsed from Node %v: %v", node.Name, shares_float)

		renewables[node.Name] = shares_float
	}

	return renewables
}

func sum(slice []float64) float64 {
	result := 0.0
	for _, v := range slice {
		result += v
	}
	return result
}

func normalizeScores(weightedScores map[string]float64) map[string]int {

	highest := 1.0
	normalizedNodeScores := make(map[string]int)

	// find highest of current shares
	for _, score := range weightedScores {
		highest = math.Max(highest, score)
	}

	// calculate normalized score for each share per node
	for node, score := range weightedScores {
		normalizedNodeScores[node] = int((score * MAX_SCORE / highest) + 0.5)
	}

	return normalizedNodeScores
}

func weightScores(nodeScores map[string][]float64, mode string, weight string) map[string]int {

	var weights []float64
	var weightConstant, _ = strconv.ParseFloat(weight, 64)
	weightedNodeScores := make(map[string]float64)

	// exponential function to distribute weights to each renewable measurement
	for i := 0.0; i < 5; i++ {
		weights = append(weights, math.Pow(weightConstant, i)/(1-weightConstant))
	}

	// favor-present is default
	if mode != "favor-future" {
		sort.Sort(sort.Reverse(sort.Float64Slice(weights)))
	}

	for node, scores := range nodeScores {

		for i := 0; i < len(scores); i++ {

			scores[i] *= weights[i]
		}
		weightedNodeScores[node] = sum(scores)
	}

	return normalizeScores(weightedNodeScores)
}

func calculateRenewableScores(nodeShares map[string][]float64) map[string][]float64 {

	normalizedScores := make(map[string][]float64)

	// go through all five renewable shares (10m, 1h, 4h, 12h, 24h) of each node
	for currentShare := 0; currentShare < 5; currentShare++ {

		highest := 1.0
		var normalizedScore float64

		// find highest of current shares
		for node, _ := range nodeShares {
			highest = math.Max(highest, nodeShares[node][currentShare])
		}

		// calculate normalized score for each share per node
		for node, shares := range nodeShares {
			normalizedScore = shares[currentShare] * MAX_SCORE / highest
			normalizedScores[node] = append(normalizedScores[node], normalizedScore)
		}
	}

	// Logs for Debugging
	for key, value := range normalizedScores {
		log.Printf("Renewable share scores calculated for Node %v: %v", key, value)
	}

	return normalizedScores
}

func calculateScoresFromRenewables(nodeList *v1.NodeList) map[string]int {

	// default
	if mode == "" {
		mode = "favor-present"
	}
	if weight == "" {
		weight = "0.75"
	}

	log.Printf("Scheduling mode %v with a weight constant of %v", mode, weight)

	var nodeShares = parseRenewablesFromNodes(nodeList)
	var nodeScores = calculateRenewableScores(nodeShares)
	var weightedTotalScores = weightScores(nodeScores, mode, weight)

	// Logs for Debugging
	for key, value := range weightedTotalScores {
		log.Printf("Total weighted score calculated for Node %v: %v", key, value)
	}

	return weightedTotalScores
}

package main

import (
	"log"
	"math"
	"sort"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
)

const MAX_SCORE = 10.0

// float between 0-1: smaller = steeply sloping weights; bigger = more even weights
const WEIGHT_CONSTANT = 0.75 // results in weights: [4 3 2.25 1.6875 1.265625]

func parseRenewablesFromNodes(nodeList *v1.NodeList) map[string][]float64 {

	renewables := make(map[string][]float64)

	// read renewable shares from node annotations
	for _, node := range nodeList.Items {

		shares_string := node.Annotations["renewable"]
		if shares_string != "" {
			log.Printf("Error parsing renewable share from node %v: No values found. Assigning a renewable energy share of 0.", node.Name)
			shares_string = "0.0"
		}

		// split string into slice with single values
		shares := strings.Split(shares_string, ";")
		var shares_float []float64

		// convert strings to floats
		for i := 0; i < len(shares); i += 1 {
			f64, _ := strconv.ParseFloat(shares[i], 64)
			shares_float[i] = float64(f64)
		}

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

func weightScores(nodeScores map[string][]float64) map[string]int {

	var weights []float64
	var short_running_app bool = true
	weightedNodeScores := make(map[string]float64)

	// exponential function to distribute weights to each renewable measurement
	for i := 0.0; i < 5; i++ {
		weights = append(weights, math.Pow(WEIGHT_CONSTANT, i)/(1-WEIGHT_CONSTANT))
	}

	if short_running_app {
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

	return normalizedScores
}

func calculateScoresFromRenewables(nodeList *v1.NodeList) map[string]int {

	var nodeShares = parseRenewablesFromNodes(nodeList)
	var nodeScores = calculateRenewableScores(nodeShares)
	var weightedTotalScores = weightScores(nodeScores)

	return weightedTotalScores
}

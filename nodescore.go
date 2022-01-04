package main

import (
	"context"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"

	v1 "k8s.io/api/core/v1"
)

// score and node constants
const MAX_SCORE = 10.0
const RATED_POWER_NODE = 10000

// read mode which was set in the scheduler.yaml
var mode = strings.ToLower(os.Getenv("MODE"))

func parseDataFromNodes(nodeList *v1.NodeList, windowSize int) map[string][]float64 {

	nodeEnergyData := make(map[string][]float64)

	// read renewable shares from node annotations
	for _, node := range nodeList.Items {

		var energyData []float64

		sharesString := node.Annotations["renewables"]
		if sharesString == "" {
			log.Printf("Error parsing renewable share from node %v: No values found. Assigning renewable energy shares of 0.", node.Name)
			for i := 0; i < windowSize; i += 1 {
				energyData = append(energyData, 0.0)
			}
		} else {
			// split renewable string into slice with single values
			shares := strings.Split(sharesString, ";")
			// convert strings to floats and append to data
			for i := 0; i < windowSize; i += 1 {
				f64, _ := strconv.ParseFloat(shares[i], 64)
				energyData = append(energyData, float64(f64))
			}
		}

		log.Printf("Renewable shares parsed from node %v: %v", node.Name, energyData)

		nodeEnergyData[node.Name] = energyData
	}

	return nodeEnergyData
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
	weightedNodeScores := make(map[string]float64)
	var scoreCount int

	for _, scores := range nodeScores {
		scoreCount = len(scores)
		break
	}

	// exponential function to distribute weights to each renewable measurement
	for i := 0; i < scoreCount; i++ {
		weights = append(weights, math.Pow(0.75, float64(i))/(1-0.75))
	}

	sort.Float64s(weights)

	// apply weights to scores
	for node, scores := range nodeScores {

		for i := 0; i < scoreCount; i++ {

			scores[i] *= weights[i]
		}
		weightedNodeScores[node] = sum(scores)
	}

	return normalizeScores(weightedNodeScores)
}

func calculateRenewableScores(nodeShares map[string][]float64, windowSize int) map[string][]float64 {

	normalizedScores := make(map[string][]float64)

	// go through all renewable shares of each node
	for currentShare := 0; currentShare < windowSize; currentShare++ {

		highest := 1.0
		lowest := 0.0
		var normalizedScore float64

		// find highest of current shares
		for node := range nodeShares {
			highest = math.Max(highest, nodeShares[node][currentShare])
			lowest = math.Min(lowest, nodeShares[node][currentShare])
		}

		// calculate normalized score for each excess / deficit share per node
		for node, shares := range nodeShares {
			normalizedScore = ((shares[currentShare] - lowest) / (highest - lowest)) * MAX_SCORE
			normalizedScores[node] = append(normalizedScores[node], roundToTwoDecimals(normalizedScore))
		}
	}

	// Logs for Debugging
	for key, value := range normalizedScores {
		log.Printf("Renewable share scores calculated for Node %v: %v", key, value)
	}

	return normalizedScores
}

func calculateRenewableExcess(energyData map[string][]float64, currentUtilization map[string]float64) map[string][]float64 {

	renewablesExcess := make(map[string][]float64)

	for node := range energyData {

		var nodeRenewableExcess []float64
		var currentNodeUtilization = currentUtilization[node]

		// calculate consumption and round to two decimal places
		var currentInput = roundToTwoDecimals(RATED_POWER_NODE * currentNodeUtilization)

		log.Printf("Node %v with max input of %v W and current utilization of %v %% has a current consumption of %v W", node, RATED_POWER_NODE, math.Round(currentNodeUtilization*1000)/10, currentInput)

		// calculate renewable energy excess for current node utilization
		for _, renewableShare := range energyData[node] {
			nodeRenewableExcess = append(nodeRenewableExcess, roundToTwoDecimals(renewableShare-currentInput))
		}

		log.Printf("Node %v has a renewable energy excess share of: %v ", node, nodeRenewableExcess)

		renewablesExcess[node] = nodeRenewableExcess
	}

	return renewablesExcess
}

func roundToTwoDecimals(input float64) float64 {
	return math.Round(input*100) / 100
}

func calculateCpuUtilization(nodeList *v1.NodeList) map[string]float64 {

	nodeUtilization := make(map[string]float64)
	var kubeconfig, master string //empty, assuming inClusterConfig

	// initiate connection to metrics server
	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		panic(err)
	}

	mc, err := metrics.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	for _, node := range nodeList.Items {

		// get node metrics from metrics server
		nodeMetricsList, err := mc.MetricsV1beta1().NodeMetricses().Get(context.TODO(), node.Name, metav1.GetOptions{})
		if err != nil {
			panic(err)
		}

		// get total allocatable CPU from node status
		cpuAllocatableCores, _ := strconv.ParseFloat(node.Status.Allocatable.Cpu().String(), 64)
		var cpuAllocatableNanoCores = cpuAllocatableCores * math.Pow(10, 9)

		// get current CPU utilization from node metrics
		cpuCurrentUsageNanoCores, _ := strconv.ParseFloat(strings.TrimSuffix(nodeMetricsList.Usage.Cpu().String(), "n"), 64)

		var totalUtilization = cpuCurrentUsageNanoCores / cpuAllocatableNanoCores

		nodeUtilization[node.Name] = totalUtilization
	}

	return nodeUtilization
}

func calculateScoresFromRenewables(nodeList *v1.NodeList) map[string]int {

	var windowSize int

	// set windows size for renewable energy (forecast) values to be considered
	switch mode {
	case "s":
		windowSize = 2
	case "m":
		windowSize = 5
	case "l":
		windowSize = 13
	case "xl":
		windowSize = 25
	default:
		windowSize = 1
	}

	// get data from annotations
	var energyData = parseDataFromNodes(nodeList, windowSize)

	// calculate current energy consumption per node
	var currentUtilization = calculateCpuUtilization(nodeList)

	// calculate renewable energy excess per node
	var renewableExcess = calculateRenewableExcess(energyData, currentUtilization)

	// calculate scores per renewable data period
	var nodeScores = calculateRenewableScores(renewableExcess, windowSize)

	// calculate total score per node
	var weightedTotalScores = weightScores(nodeScores)

	for key, value := range weightedTotalScores {
		log.Printf("Total weighted score calculated for Node %v: %v", key, value)
	}

	return weightedTotalScores
}

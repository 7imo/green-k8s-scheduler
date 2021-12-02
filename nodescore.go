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

const MAX_SCORE = 10.0

// reads envs from deployment
var mode = os.Getenv("MODE")
var weight = os.Getenv("WEIGHT")

func parseDataFromNodes(nodeList *v1.NodeList) map[string][]float64 {

	nodeEnergyData := make(map[string][]float64)

	// read renewable shares from node annotations
	for _, node := range nodeList.Items {

		var energyData []float64

		nominalMax := node.Annotations["nominal_power"]
		if nominalMax == "" {
			// this value is known in real setups
			log.Printf("Error parsing max nominal power from node %v: No values found. Assigning a nominal power of 10 kW.", node.Name)
			nominalMax = "10000"
		}

		// append nominal power
		nmf64, _ := strconv.ParseFloat(nominalMax, 64)
		energyData = append(energyData, float64(nmf64))

		sharesString := node.Annotations["renewable"]
		if sharesString == "" {
			log.Printf("Error parsing renewable share from node %v: No values found. Assigning a renewable energy share of 0.", node.Name)
			sharesString = "0.0;0.0;0.0;0.0;0.0"
		}

		// split renewable string into slice with single values
		shares := strings.Split(sharesString, ";")
		// convert strings to floats and append to data
		for i := 0; i < len(shares); i += 1 {
			f64, _ := strconv.ParseFloat(shares[i], 64)
			energyData = append(energyData, float64(f64))
		}

		// Logs for Debugging
		log.Printf("Nominal Power and Renewable shares parsed from Node %v: %v", node.Name, energyData)

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

func calculateRenewableSurpas(energyData map[string][]float64, currentUtilization map[string]float64) map[string][]float64 {

	renewablesSurpas := make(map[string][]float64)

	for node := range energyData {

		var maxInput float64
		var nodeRenewableSurpas []float64
		var currentNodeUtilization = currentUtilization[node]
		var currentInput = maxInput * currentNodeUtilization

		// split nominal power and renewable shares
		maxInput, energyData[node] = energyData[node][0], energyData[node][1:]

		log.Printf("Node %v with max input of %v W and current utilization of %v %% has a current input of %v W", node, maxInput, currentNodeUtilization*10, currentInput)

		// calculate renewable energy surpas for current node utilization
		for _, renewableShare := range energyData[node] {
			renewableShare -= currentInput

			if renewableShare > 0 {
				nodeRenewableSurpas = append(nodeRenewableSurpas, renewableShare)
			} else {
				nodeRenewableSurpas = append(nodeRenewableSurpas, 0)
			}
		}

		log.Printf("Node %v has a surpas renewable energy share of: %v ", node, nodeRenewableSurpas)

		renewablesSurpas[node] = nodeRenewableSurpas
	}

	return renewablesSurpas
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

		var totalUtilization = math.Round(cpuCurrentUsageNanoCores/cpuAllocatableNanoCores*100) / 100

		nodeUtilization[node.Name] = totalUtilization
	}

	return nodeUtilization
}

func calculateScoresFromRenewables(nodeList *v1.NodeList) map[string]int {

	// default
	if mode == "" {
		mode = "favor-present"
	}
	if weight == "" {
		weight = "0.75"
	}

	log.Printf("Scheduling mode %v with a weight distribution constant of %v", mode, weight)

	var energyData = parseDataFromNodes(nodeList)
	var currentUtilization = calculateCpuUtilization(nodeList)
	var renewableSurpas = calculateRenewableSurpas(energyData, currentUtilization)
	var nodeScores = calculateRenewableScores(renewableSurpas)
	var weightedTotalScores = weightScores(nodeScores, mode, weight)

	for key, value := range weightedTotalScores {
		log.Printf("Total weighted score calculated for Node %v: %v", key, value)
	}

	return weightedTotalScores
}

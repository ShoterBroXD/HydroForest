package forest

import (
	"math"
	"math/rand"
	"sort"
)

type Config struct {
	NumTrees        int
	MaxDepth        int
	MinSamplesSplit int
	NumClasses      int
	Seed            int64
	Verbose         bool // si es true, imprime cuando cada árbol empieza y termina
}

type Node struct {
	IsLeaf     bool
	ClassLabel int
	FeatureIdx int
	Threshold  float64
	Left       *Node
	Right      *Node
}

func giniImpurity(counts []int, total int) float64 {
	if total == 0 {
		return 0
	}
	sum := 0.0
	for _, c := range counts {
		p := float64(c) / float64(total)
		sum += p * p
	}
	return 1 - sum
}

func randomFeatureSubset(numFeatures int, rng *rand.Rand) []int {
	m := int(math.Sqrt(float64(numFeatures)))
	if m < 1 {
		m = 1
	}
	all := rng.Perm(numFeatures)
	return all[:m]
}

func bestSplit(X [][]float64, y []int, indices []int, candidateFeatures []int, numClasses int) (featureIdx int, threshold float64, gain float64, found bool) {
	n := len(indices)
	totalCounts := make([]int, numClasses)
	for _, i := range indices {
		totalCounts[y[i]]++
	}
	parentGini := giniImpurity(totalCounts, n)

	bestGain := -1.0
	bestFeature := -1
	bestThreshold := 0.0

	order := make([]int, n)
	for _, f := range candidateFeatures {
		copy(order, indices)
		sort.Slice(order, func(a, b int) bool { return X[order[a]][f] < X[order[b]][f] })

		leftCounts := make([]int, numClasses)
		rightCounts := make([]int, numClasses)
		copy(rightCounts, totalCounts)

		for i := 0; i < n-1; i++ {
			cls := y[order[i]]
			leftCounts[cls]++
			rightCounts[cls]--

			if X[order[i]][f] == X[order[i+1]][f] {
				continue
			}
			nLeft := i + 1
			nRight := n - nLeft
			weighted := (float64(nLeft)/float64(n))*giniImpurity(leftCounts, nLeft) +
				(float64(nRight)/float64(n))*giniImpurity(rightCounts, nRight)
			gain := parentGini - weighted
			if gain > bestGain {
				bestGain = gain
				bestFeature = f
				bestThreshold = (X[order[i]][f] + X[order[i+1]][f]) / 2
			}
		}
	}

	if bestFeature == -1 {
		return 0, 0, 0, false
	}
	return bestFeature, bestThreshold, bestGain, true
}

func majorityClass(y []int, indices []int, numClasses int) int {
	counts := make([]int, numClasses)
	for _, i := range indices {
		counts[y[i]]++
	}
	best, bestCount := 0, -1
	for c, cnt := range counts {
		if cnt > bestCount {
			best, bestCount = c, cnt
		}
	}
	return best
}

func buildTree(X [][]float64, y []int, indices []int, depth int, cfg Config, rng *rand.Rand) *Node {
	if depth >= cfg.MaxDepth || len(indices) < cfg.MinSamplesSplit || isPure(y, indices) {
		return &Node{IsLeaf: true, ClassLabel: majorityClass(y, indices, cfg.NumClasses)}
	}

	candidates := randomFeatureSubset(len(X[0]), rng)
	featureIdx, threshold, gain, found := bestSplit(X, y, indices, candidates, cfg.NumClasses)
	if !found || gain <= 0 {
		return &Node{IsLeaf: true, ClassLabel: majorityClass(y, indices, cfg.NumClasses)}
	}

	var leftIdx, rightIdx []int
	for _, i := range indices {
		if X[i][featureIdx] <= threshold {
			leftIdx = append(leftIdx, i)
		} else {
			rightIdx = append(rightIdx, i)
		}
	}
	if len(leftIdx) == 0 || len(rightIdx) == 0 {
		return &Node{IsLeaf: true, ClassLabel: majorityClass(y, indices, cfg.NumClasses)}
	}

	return &Node{
		IsLeaf:     false,
		FeatureIdx: featureIdx,
		Threshold:  threshold,
		Left:       buildTree(X, y, leftIdx, depth+1, cfg, rng),
		Right:      buildTree(X, y, rightIdx, depth+1, cfg, rng),
	}
}

func isPure(y []int, indices []int) bool {
	if len(indices) == 0 {
		return true
	}
	first := y[indices[0]]
	for _, i := range indices[1:] {
		if y[i] != first {
			return false
		}
	}
	return true
}

func predictTree(node *Node, x []float64) int {
	for !node.IsLeaf {
		if x[node.FeatureIdx] <= node.Threshold {
			node = node.Left
		} else {
			node = node.Right
		}
	}
	return node.ClassLabel
}

func bootstrapSample(n int, rng *rand.Rand) []int {
	idx := make([]int, n)
	for i := range idx {
		idx[i] = rng.Intn(n)
	}
	return idx
}

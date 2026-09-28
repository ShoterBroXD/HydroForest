package forest

type Forest struct {
	Trees      []*Node
	NumClasses int
}

func (f *Forest) PredictOne(x []float64) int {
	votes := make([]int, f.NumClasses)
	for _, t := range f.Trees {
		votes[predictTree(t, x)]++
	}
	best, bestCount := 0, -1
	for c, v := range votes {
		if v > bestCount {
			best, bestCount = c, v
		}
	}
	return best
}

func (f *Forest) Accuracy(X [][]float64, y []int) float64 {
	if len(X) == 0 {
		return 0
	}
	correct := 0
	for i, x := range X {
		if f.PredictOne(x) == y[i] {
			correct++
		}
	}
	return float64(correct) / float64(len(X))
}

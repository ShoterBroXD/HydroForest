package forest

import "math/rand"

func TrainSequential(X [][]float64, y []int, cfg Config) *Forest {
	trees := make([]*Node, cfg.NumTrees)
	rng := rand.New(rand.NewSource(cfg.Seed))
	n := len(X)

	for i := 0; i < cfg.NumTrees; i++ {
		sampleIdx := bootstrapSample(n, rng)
		trees[i] = buildTree(X, y, sampleIdx, 0, cfg, rng)
	}

	return &Forest{Trees: trees, NumClasses: cfg.NumClasses}
}

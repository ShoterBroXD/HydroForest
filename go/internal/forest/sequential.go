package forest

import (
	"fmt"
	"math/rand"
	"time"
)

func TrainSequential(X [][]float64, y []int, cfg Config) *Forest {
	trees := make([]*Node, cfg.NumTrees)
	rng := rand.New(rand.NewSource(cfg.Seed))
	n := len(X)
	start := time.Now()

	for i := 0; i < cfg.NumTrees; i++ {
		if cfg.Verbose {
			fmt.Printf("[t=%7.2fs] hilo principal toma el árbol %02d\n", time.Since(start).Seconds(), i)
		}
		treeStart := time.Now()

		sampleIdx := bootstrapSample(n, rng)
		trees[i] = buildTree(X, y, sampleIdx, 0, cfg, rng)

		if cfg.Verbose {
			fmt.Printf("[t=%7.2fs] hilo principal termina el árbol %02d (%.2f s)\n",
				time.Since(start).Seconds(), i, time.Since(treeStart).Seconds())
		}
	}

	return &Forest{Trees: trees, NumClasses: cfg.NumClasses}
}

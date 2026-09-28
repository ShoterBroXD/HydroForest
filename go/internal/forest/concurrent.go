package forest

import (
	"math/rand"
	"sync"
)

func TrainConcurrent(X [][]float64, y []int, cfg Config, numWorkers int) *Forest {
	n := len(X)

	jobs := make(chan int, cfg.NumTrees)
	for i := 0; i < cfg.NumTrees; i++ {
		jobs <- i
	}

	close(jobs)
	var mu sync.Mutex
	var wg sync.WaitGroup
	trees := make([]*Node, 0, cfg.NumTrees)
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(cfg.Seed + int64(workerID) + 1))

			for range jobs {
				sampleIdx := bootstrapSample(n, rng)
				tree := buildTree(X, y, sampleIdx, 0, cfg, rng)

				mu.Lock()
				trees = append(trees, tree)
				mu.Unlock()
			}
		}(w)
	}

	wg.Wait()

	return &Forest{Trees: trees, NumClasses: cfg.NumClasses}
}

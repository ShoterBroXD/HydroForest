package forest

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

func TrainConcurrent(X [][]float64, y []int, cfg Config, numWorkers int) *Forest {
	n := len(X)
	start := time.Now()

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

			for treeIdx := range jobs {
				if cfg.Verbose {
					fmt.Printf("[t=%7.2fs] worker %02d toma el árbol %02d\n",
						time.Since(start).Seconds(), workerID, treeIdx)
				}
				treeStart := time.Now()

				sampleIdx := bootstrapSample(n, rng)
				tree := buildTree(X, y, sampleIdx, 0, cfg, rng)

				mu.Lock()
				trees = append(trees, tree)
				mu.Unlock()

				if cfg.Verbose {
					fmt.Printf("[t=%7.2fs] worker %02d termina el árbol %02d (%.2f s)\n",
						time.Since(start).Seconds(), workerID, treeIdx, time.Since(treeStart).Seconds())
				}
			}
		}(w)
	}

	wg.Wait()

	return &Forest{Trees: trees, NumClasses: cfg.NumClasses}
}

package bench

import (
	"fmt"
	"runtime"
	"sort"
	"time"

	"hydroforest/internal/forest"
)

func TrimmedMean(values []float64, trim int) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	if 2*trim >= len(sorted) {
		trim = 0
	}
	sorted = sorted[trim : len(sorted)-trim]
	sum := 0.0
	for _, v := range sorted {
		sum += v
	}
	return sum / float64(len(sorted))
}

type RunResult struct {
	Label      string
	NumWorkers int
	Times      []float64
	TrimmedAvg float64
}

func TimeRuns(runs int, train func() *forest.Forest) []float64 {
	times := make([]float64, runs)
	for i := 0; i < runs; i++ {
		start := time.Now()
		train()
		times[i] = time.Since(start).Seconds()
	}
	return times
}

func SpeedupTable(X [][]float64, y []int, cfg forest.Config, numWorkers, runs, trim int) (seq, con RunResult, speedup float64) {
	seqTimes := TimeRuns(runs, func() *forest.Forest {
		return forest.TrainSequential(X, y, cfg)
	})
	conTimes := TimeRuns(runs, func() *forest.Forest {
		return forest.TrainConcurrent(X, y, cfg, numWorkers)
	})

	seq = RunResult{Label: "Secuencial", NumWorkers: 1, Times: seqTimes, TrimmedAvg: TrimmedMean(seqTimes, trim)}
	con = RunResult{Label: "Concurrente", NumWorkers: numWorkers, Times: conTimes, TrimmedAvg: TrimmedMean(conTimes, trim)}
	speedup = seq.TrimmedAvg / con.TrimmedAvg
	return
}

type ScalabilityRow struct {
	Workers    int
	TrimmedAvg float64
	Speedup    float64
	Efficiency float64
}

func ScalabilitySweep(X [][]float64, y []int, cfg forest.Config, workerCounts []int, runs, trim int, baselineSeq float64) []ScalabilityRow {
	rows := make([]ScalabilityRow, 0, len(workerCounts))
	for _, w := range workerCounts {
		times := TimeRuns(runs, func() *forest.Forest {
			return forest.TrainConcurrent(X, y, cfg, w)
		})
		avg := TrimmedMean(times, trim)
		speedup := baselineSeq / avg
		rows = append(rows, ScalabilityRow{
			Workers:    w,
			TrimmedAvg: avg,
			Speedup:    speedup,
			Efficiency: speedup / float64(w),
		})
	}
	return rows
}

type ResourceSnapshot struct {
	NumCPU       int
	GOMAXPROCS   int
	NumGoroutine int
	HeapAllocMB  float64
	TotalAllocMB float64
	NumGC        uint32
}

func CaptureResources() ResourceSnapshot {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return ResourceSnapshot{
		NumCPU:       runtime.NumCPU(),
		GOMAXPROCS:   runtime.GOMAXPROCS(0),
		NumGoroutine: runtime.NumGoroutine(),
		HeapAllocMB:  float64(m.HeapAlloc) / (1024 * 1024),
		TotalAllocMB: float64(m.TotalAlloc) / (1024 * 1024),
		NumGC:        m.NumGC,
	}
}

func PrintSpeedupTable(seq, con RunResult, speedup float64) {
	fmt.Println()
	fmt.Println("=== Speedup: secuencial vs. concurrente ===")
	fmt.Printf("%-14s %-10s %-14s %s\n", "Versión", "Workers", "Media recortada", "Tiempos individuales (s)")
	fmt.Printf("%-14s %-10d %-14.4f %v\n", seq.Label, seq.NumWorkers, seq.TrimmedAvg, roundAll(seq.Times))
	fmt.Printf("%-14s %-10d %-14.4f %v\n", con.Label, con.NumWorkers, con.TrimmedAvg, roundAll(con.Times))
	fmt.Printf("\nSpeedup (T-Secuencial / T-Concurrente) = %.2fx\n", speedup)
}

func PrintScalabilityTable(rows []ScalabilityRow) {
	fmt.Println()
	fmt.Println("=== Escalabilidad por número de workers ===")
	fmt.Printf("%-10s %-16s %-12s %s\n", "Workers", "Media recortada", "Speedup", "Eficiencia (speedup/worker)")
	for _, r := range rows {
		fmt.Printf("%-10d %-16.4f %-12.2f %.2f\n", r.Workers, r.TrimmedAvg, r.Speedup, r.Efficiency)
	}
}

func PrintResources(before, after ResourceSnapshot) {
	fmt.Println()
	fmt.Println("=== Uso de recursos de cómputo ===")
	fmt.Printf("CPUs lógicos disponibles (NumCPU): %d\n", after.NumCPU)
	fmt.Printf("GOMAXPROCS: %d\n", after.GOMAXPROCS)
	fmt.Printf("Goroutines activas al terminar: %d\n", after.NumGoroutine)
	fmt.Printf("Heap en uso: %.2f MB (antes: %.2f MB)\n", after.HeapAllocMB, before.HeapAllocMB)
	fmt.Printf("Memoria total reservada acumulada: %.2f MB\n", after.TotalAllocMB)
	fmt.Printf("Ciclos de garbage collection ejecutados: %d\n", after.NumGC-before.NumGC)
}

func roundAll(v []float64) []float64 {
	out := make([]float64, len(v))
	for i, x := range v {
		out[i] = float64(int(x*10000)) / 10000
	}
	return out
}

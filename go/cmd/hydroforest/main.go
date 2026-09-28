package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"

	"hydroforest/internal/bench"
	"hydroforest/internal/dataset"
	"hydroforest/internal/forest"
)

func main() {
	dataPath := flag.String("data", "", "ruta al CSV limpio (water_quality_clean.csv)")
	numTrees := flag.Int("trees", 60, "número de árboles del Random Forest")
	maxDepth := flag.Int("max-depth", 12, "profundidad máxima por árbol")
	minSplit := flag.Int("min-samples-split", 20, "mínimo de muestras para dividir un nodo")
	runs := flag.Int("runs", 5, "número de corridas por configuración, para la media recortada")
	trim := flag.Int("trim", 1, "cuántas corridas se recortan de cada extremo (media recortada)")
	outCSV := flag.String("out", "results.csv", "archivo CSV donde guardar la tabla de speedup")
	flag.Parse()

	fmt.Println("=== HydroForest - PC2: Random Forest secuencial vs. concurrente ===")

	if *dataPath == "" {
		fmt.Fprintln(os.Stderr, "falta el dataset: usa -data ruta/a/water_quality_clean.csv")
		os.Exit(1)
	}
	fmt.Printf("Cargando dataset desde %s ...\n", *dataPath)
	ds, err := dataset.LoadCSV(*dataPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error cargando el dataset: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Dataset listo: %d filas, %d features, %d clases (%v)\n",
		ds.Len(), len(dataset.FeatureNames), len(dataset.ClassNames), dataset.ClassNames)

	train, test := dataset.Split(ds, 0.8, 7)
	fmt.Printf("Split: %d train / %d test\n", train.Len(), test.Len())

	cfg := forest.Config{
		NumTrees:        *numTrees,
		MaxDepth:        *maxDepth,
		MinSamplesSplit: *minSplit,
		NumClasses:      len(dataset.ClassNames),
		Seed:            42,
	}
	numWorkers := runtime.NumCPU()
	fmt.Printf("Config: %d árboles, profundidad máx %d, %d corridas por versión, %d workers (NumCPU)\n",
		cfg.NumTrees, cfg.MaxDepth, *runs, numWorkers)

	resBefore := bench.CaptureResources()

	// 1) Speedup: secuencial vs. concurrente (media recortada)
	seq, con, speedup := bench.SpeedupTable(train.X, train.Y, cfg, numWorkers, *runs, *trim)
	bench.PrintSpeedupTable(seq, con, speedup)

	resAfter := bench.CaptureResources()
	bench.PrintResources(resBefore, resAfter)

	// 2) Escalabilidad: variar el número de workers
	workerCounts := scalabilityCounts(numWorkers)
	rows := bench.ScalabilitySweep(train.X, train.Y, cfg, workerCounts, *runs, *trim, seq.TrimmedAvg)
	bench.PrintScalabilityTable(rows)

	// 3) Accuracy: confirmar que ambas versiones aprenden lo mismo
	fSeq := forest.TrainSequential(train.X, train.Y, cfg)
	fCon := forest.TrainConcurrent(train.X, train.Y, cfg, numWorkers)
	fmt.Println()
	fmt.Println("=== Accuracy en test (equivalencia funcional seq vs. concurrente) ===")
	fmt.Printf("Secuencial:  %.2f%%\n", fSeq.Accuracy(test.X, test.Y)*100)
	fmt.Printf("Concurrente: %.2f%%\n", fCon.Accuracy(test.X, test.Y)*100)

	if err := writeResultsCSV(*outCSV, seq, con, speedup, rows); err != nil {
		fmt.Fprintf(os.Stderr, "no se pudo guardar %s: %v\n", *outCSV, err)
	} else {
		fmt.Printf("\nTabla de resultados guardada en %s\n", *outCSV)
	}
}

func scalabilityCounts(numCPU int) []int {
	counts := []int{1}
	for w := 2; w < numCPU; w *= 2 {
		counts = append(counts, w)
	}
	if counts[len(counts)-1] != numCPU {
		counts = append(counts, numCPU)
	}
	return counts
}

func writeResultsCSV(path string, seq, con bench.RunResult, speedup float64, rows []bench.ScalabilityRow) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"seccion", "workers", "media_recortada_s", "speedup", "eficiencia"})
	w.Write([]string{"secuencial", strconv.Itoa(seq.NumWorkers), fmt.Sprintf("%.4f", seq.TrimmedAvg), "1.00", "1.00"})
	w.Write([]string{"concurrente", strconv.Itoa(con.NumWorkers), fmt.Sprintf("%.4f", con.TrimmedAvg), fmt.Sprintf("%.2f", speedup), fmt.Sprintf("%.2f", speedup/float64(con.NumWorkers))})
	for _, r := range rows {
		w.Write([]string{"escalabilidad", strconv.Itoa(r.Workers), fmt.Sprintf("%.4f", r.TrimmedAvg), fmt.Sprintf("%.2f", r.Speedup), fmt.Sprintf("%.2f", r.Efficiency)})
	}
	return nil
}

package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

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
	mode := flag.String("mode", "bench", "bench: speedup y escalabilidad | demo: una corrida con registro por worker | sweep: barrido fino de 1 a N workers | predict: evaluador interactivo para usuarios")
	demoWorkers := flag.Int("workers", 0, "workers para el modo demo (0 = NumCPU)")
	baseline := flag.Float64("baseline", 0, "modo sweep: tiempo secuencial ya medido en segundos (0 = medirlo)")
	threshold := flag.Float64("threshold", 5, "modo sweep: ganancia marginal mínima (%) para considerar que un worker extra vale la pena")
	sweepOut := flag.String("sweep-out", "sweep.csv", "modo sweep: archivo CSV del barrido fino")
	modelPath := flag.String("model", "hydroforest_model.gob", "modo predict: archivo donde se guarda el modelo entrenado")
	flag.Parse()

	if *mode == "predict" {
		if *dataPath == "" {
			fmt.Fprintln(os.Stderr, "falta el dataset: usa -data ruta/a/water_quality_clean.csv")
			os.Exit(1)
		}
		runPredict(*dataPath, *modelPath)
		return
	}

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

	switch *mode {
	case "demo":
		runDemo(train, test, cfg, *demoWorkers)
		return
	case "sweep":
		runSweep(train, cfg, numWorkers, *runs, *trim, *baseline, *threshold, *sweepOut)
		return
	case "bench":
	default:
		fmt.Fprintf(os.Stderr, "modo desconocido %q (usa bench, demo, sweep o predict)\n", *mode)
		os.Exit(1)
	}

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

// runDemo entrena una vez cada versión mostrando qué árbol toma y termina cada
// worker. Sirve como evidencia de ejecución (no se usa para medir speedup).
func runDemo(train, test *dataset.Dataset, cfg forest.Config, workers int) {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	cfg.Verbose = true

	fmt.Println()
	fmt.Println("=== DEMO secuencial (1 hilo) ===")
	t0 := time.Now()
	fSeq := forest.TrainSequential(train.X, train.Y, cfg)
	tSeq := time.Since(t0).Seconds()
	fmt.Printf("Tiempo secuencial: %.2f s\n", tSeq)

	fmt.Println()
	fmt.Printf("=== DEMO concurrente (%d workers) ===\n", workers)
	t0 = time.Now()
	fCon := forest.TrainConcurrent(train.X, train.Y, cfg, workers)
	tCon := time.Since(t0).Seconds()
	fmt.Printf("Tiempo concurrente: %.2f s\n", tCon)

	fmt.Println()
	fmt.Println("=== Resumen de la demo ===")
	fmt.Printf("Speedup de esta corrida: %.2fx\n", tSeq/tCon)
	fmt.Printf("Accuracy secuencial:  %.2f%%\n", fSeq.Accuracy(test.X, test.Y)*100)
	fmt.Printf("Accuracy concurrente: %.2f%%\n", fCon.Accuracy(test.X, test.Y)*100)
}

// runSweep mide el tiempo con 1, 2, 3, ... maxWorkers workers y ubica el
// punto de equilibrio. Guarda la tabla completa en un CSV.
func runSweep(train *dataset.Dataset, cfg forest.Config, maxWorkers, runs, trim int, baseline, threshold float64, out string) {
	if baseline <= 0 {
		fmt.Printf("\nMidiendo la línea base secuencial (%d corridas)...\n", runs)
		seqTimes := bench.TimeRuns(runs, func() *forest.Forest {
			return forest.TrainSequential(train.X, train.Y, cfg)
		})
		baseline = bench.TrimmedMean(seqTimes, trim)
	}
	fmt.Printf("Línea base secuencial: %.2f s\n", baseline)

	rows := bench.FineSweep(train.X, train.Y, cfg, maxWorkers, runs, trim, baseline)
	eq := bench.MarkEquilibrium(rows, threshold)

	fmt.Println()
	if eq > 0 {
		fmt.Printf("Punto de equilibrio: %d workers (agregar uno más reduce el tiempo en menos de %.1f%%)\n", eq, threshold)
	} else {
		fmt.Printf("No se alcanzó el punto de equilibrio: cada worker agregado siguió reduciendo el tiempo en %.1f%% o más\n", threshold)
	}

	if err := writeSweepCSV(out, baseline, rows); err != nil {
		fmt.Fprintf(os.Stderr, "no se pudo guardar %s: %v\n", out, err)
	} else {
		fmt.Printf("Barrido guardado en %s\n", out)
	}
}

func writeSweepCSV(path string, baseline float64, rows []bench.FineSweepRow) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"workers", "media_recortada_s", "speedup", "eficiencia", "ganancia_marginal_pct", "equilibrio", "baseline_secuencial_s"})
	for i, r := range rows {
		gain := ""
		if i > 0 {
			gain = fmt.Sprintf("%.2f", r.MarginalGain)
		}
		eq := "0"
		if r.IsEquilibrium {
			eq = "1"
		}
		w.Write([]string{
			strconv.Itoa(r.Workers),
			fmt.Sprintf("%.4f", r.TrimmedAvg),
			fmt.Sprintf("%.2f", r.Speedup),
			fmt.Sprintf("%.2f", r.Efficiency),
			gain,
			eq,
			fmt.Sprintf("%.4f", baseline),
		})
	}
	return nil
}

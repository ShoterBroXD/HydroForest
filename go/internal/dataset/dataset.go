package dataset

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
)

var FeatureNames = []string{
	"pH_ph_units",
	"Dissolved_Oxygen_mgl",
	"Biochemical_Oxygen_Demand_mgl",
	"Ammonia_mgl",
	"Nitrate_mgl",
	"Orthophosphate_mgl",
	"Nitrogen_mgl",
	"Temperature_cel",
}

const TargetName = "CCME_WQI"

var ClassNames = []string{"Poor", "Marginal", "Fair", "Good", "Excellent"}

var classIndex = map[string]int{
	"Poor":      0,
	"Marginal":  1,
	"Fair":      2,
	"Good":      3,
	"Excellent": 4,
}

type Dataset struct {
	X [][]float64
	Y []int
}

func (d *Dataset) Len() int { return len(d.X) }

func LoadCSV(path string) (*Dataset, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.ReuseRecord = true

	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el encabezado: %w", err)
	}

	colIdx := make(map[string]int, len(header))
	for i, h := range header {
		colIdx[strings.TrimSpace(strings.TrimPrefix(h, "\uFEFF"))] = i
	}

	featureCols := make([]int, len(FeatureNames))
	for i, name := range FeatureNames {
		idx, ok := colIdx[name]
		if !ok {
			return nil, fmt.Errorf("columna requerida no encontrada en el CSV: %q", name)
		}
		featureCols[i] = idx
	}
	targetIdx, ok := colIdx[TargetName]
	if !ok {
		return nil, fmt.Errorf("columna objetivo %q no encontrada en el CSV", TargetName)
	}

	ds := &Dataset{}
	skipped := 0
	for {
		row, err := r.Read()
		if err != nil {
			break
		}

		cls, ok := classIndex[strings.TrimSpace(row[targetIdx])]
		if !ok {
			skipped++
			continue
		}

		feats := make([]float64, len(featureCols))
		valid := true
		for i, ci := range featureCols {
			v, err := strconv.ParseFloat(strings.TrimSpace(row[ci]), 64)
			if err != nil {
				valid = false
				break
			}
			feats[i] = v
		}
		if !valid {
			skipped++
			continue
		}

		ds.X = append(ds.X, feats)
		ds.Y = append(ds.Y, cls)
	}

	if skipped > 0 {
		fmt.Printf("Aviso: %d filas descartadas por valores no válidos\n", skipped)
	}
	return ds, nil
}

func Split(d *Dataset, trainFrac float64, seed int64) (train, test *Dataset) {
	n := d.Len()
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	rng := rand.New(rand.NewSource(seed))
	rng.Shuffle(n, func(i, j int) { idx[i], idx[j] = idx[j], idx[i] })

	nTrain := int(float64(n) * trainFrac)
	train = &Dataset{X: make([][]float64, nTrain), Y: make([]int, nTrain)}
	test = &Dataset{X: make([][]float64, n-nTrain), Y: make([]int, n-nTrain)}

	for i, id := range idx {
		if i < nTrain {
			train.X[i] = d.X[id]
			train.Y[i] = d.Y[id]
		} else {
			test.X[i-nTrain] = d.X[id]
			test.Y[i-nTrain] = d.Y[id]
		}
	}
	return train, test
}

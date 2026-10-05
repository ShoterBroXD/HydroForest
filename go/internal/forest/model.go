package forest

import (
	"encoding/gob"
	"fmt"
	"os"
)

// Votes devuelve cuántos árboles del bosque votaron por cada clase.
func (f *Forest) Votes(x []float64) []int {
	votes := make([]int, f.NumClasses)
	for _, t := range f.Trees {
		votes[predictTree(t, x)]++
	}
	return votes
}

// Save guarda el bosque entrenado en disco (formato gob de la librería
// estándar), para poder usarlo después sin volver a entrenar.
func (f *Forest) Save(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := gob.NewEncoder(file).Encode(f); err != nil {
		return fmt.Errorf("no se pudo guardar el modelo: %w", err)
	}
	return nil
}

// Load carga un bosque guardado previamente con Save.
func Load(path string) (*Forest, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var f Forest
	if err := gob.NewDecoder(file).Decode(&f); err != nil {
		return nil, fmt.Errorf("no se pudo leer el modelo: %w", err)
	}
	return &f, nil
}

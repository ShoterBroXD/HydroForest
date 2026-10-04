package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"hydroforest/internal/dataset"
	"hydroforest/internal/forest"
)

// medicion describe, en lenguaje simple, cada valor que se le pide al usuario.
// El orden debe coincidir con dataset.FeatureNames.
type medicion struct {
	nombre      string
	explicacion string
	unidad      string
	min, max    float64
}

var mediciones = []medicion{
	{"Acidez (pH)", "Qué tan ácida es el agua. Va de 0 a 14; el agua neutra tiene 7.", "", 0, 14},
	{"Oxígeno disuelto", "Cuánto oxígeno tiene el agua para los peces. Mientras más, mejor.", "mg/L", 0, 20},
	{"Materia orgánica (DBO)", "Cuánta materia en descomposición hay en el agua. Mientras menos, mejor.", "mg/L", 0, 1000},
	{"Amoniaco", "Suele venir de desagües o desechos de animales. Mientras menos, mejor.", "mg/L", 0, 1000},
	{"Nitratos", "Suelen venir de fertilizantes usados en el campo. Mientras menos, mejor.", "mg/L", 0, 1000},
	{"Fosfatos", "Suelen venir de detergentes y fertilizantes. Mientras menos, mejor.", "mg/L", 0, 1000},
	{"Nitrógeno", "Otro nutriente que en exceso contamina el agua. Mientras menos, mejor.", "mg/L", 0, 1000},
	{"Temperatura", "Temperatura del agua.", "°C", -5, 45},
}

// Nombres y significado de cada nivel de calidad, en el mismo orden que dataset.ClassNames.
var nivelesCalidad = []struct {
	nombre      string
	significado string
}{
	{"MALA", "El agua está contaminada casi siempre. No debería usarse sin un tratamiento previo."},
	{"MARGINAL", "El agua presenta contaminación con frecuencia. Conviene vigilarla y evitar su uso directo."},
	{"ACEPTABLE", "El agua está en un estado aceptable, aunque a veces muestra signos de contaminación."},
	{"BUENA", "El agua está en buen estado, con alteraciones mínimas."},
	{"EXCELENTE", "El agua está en muy buen estado, como la de un río o lago sano."},
}

func runPredict(dataPath, modelPath string) {
	in := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("==============================================================")
	fmt.Println("   HydroForest - Evaluador de calidad del agua")
	fmt.Println("==============================================================")
	fmt.Println("Este programa estima qué tan limpia está el agua de un río o lago")
	fmt.Println("a partir de algunas mediciones básicas de una muestra.")
	fmt.Println()

	fmt.Println("Cargando los datos de referencia, un momento...")
	ds, err := dataset.LoadCSV(dataPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "No se pudieron cargar los datos: %v\n", err)
		os.Exit(1)
	}
	train, test := dataset.Split(ds, 0.8, 7)
	tipicos := valoresTipicos(train.X)

	modelo, err := forest.Load(modelPath)
	if err != nil {
		fmt.Println("Preparando el sistema por primera vez (puede tardar alrededor de un minuto)...")
		cfg := forest.Config{NumTrees: 20, MaxDepth: 12, MinSamplesSplit: 20, NumClasses: len(dataset.ClassNames), Seed: 42}
		inicio := time.Now()
		modelo = forest.TrainConcurrent(train.X, train.Y, cfg, runtime.NumCPU())
		if err := modelo.Save(modelPath); err != nil {
			fmt.Fprintf(os.Stderr, "Aviso: no se pudo guardar el modelo (%v); se volverá a preparar la próxima vez.\n", err)
		}
		fmt.Printf("Sistema listo (preparado en %.0f segundos).\n", time.Since(inicio).Seconds())
	} else {
		fmt.Println("Sistema listo.")
	}

	for {
		fmt.Println()
		fmt.Println("--------------------------------------------------------------")
		fmt.Println("¿Qué deseas hacer?")
		fmt.Println("  1) Evaluar una muestra de agua con mis propios datos")
		fmt.Println("  2) Ver un ejemplo con una muestra real de un río o lago")
		fmt.Println("  3) Salir")
		opcion, ok := leerLinea(in, "Escribe el número de la opción y presiona Enter: ")
		if !ok {
			return
		}
		switch opcion {
		case "1":
			muestra, ok := pedirMuestra(in, tipicos)
			if !ok {
				return
			}
			mostrarResultado(modelo, muestra)
		case "2":
			i := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(test.Len())
			muestra := test.X[i]
			fmt.Println()
			fmt.Println("Muestra real tomada de los datos de monitoreo:")
			for j, m := range mediciones {
				fmt.Printf("  %-24s %8.2f %s\n", m.nombre+":", muestra[j], m.unidad)
			}
			pred := mostrarResultado(modelo, muestra)
			real := test.Y[i]
			fmt.Printf("Calidad registrada en el monitoreo real: %s\n", nivelesCalidad[real].nombre)
			if pred == real {
				fmt.Println("El sistema ACERTÓ.")
			} else {
				fmt.Println("En este caso el sistema NO acertó.")
			}
		case "3":
			fmt.Println("¡Gracias por usar HydroForest!")
			return
		default:
			fmt.Println("Opción no válida. Escribe 1, 2 o 3.")
		}
	}
}

// pedirMuestra le pide al usuario cada medición, explicándole qué es y
// ofreciéndole un valor típico si no lo conoce.
func pedirMuestra(in *bufio.Reader, tipicos []float64) ([]float64, bool) {
	fmt.Println()
	fmt.Println("Ingresa las mediciones de tu muestra de agua.")
	fmt.Println("Si no conoces algún valor, solo presiona Enter y se usará un valor típico.")
	muestra := make([]float64, len(mediciones))
	for j, m := range mediciones {
		fmt.Println()
		fmt.Printf("%d. %s\n", j+1, m.nombre)
		fmt.Printf("   %s\n", m.explicacion)
		for {
			pregunta := fmt.Sprintf("   Valor %s[Enter = %.2f]: ", unidadEntre(m.unidad), tipicos[j])
			texto, ok := leerLinea(in, pregunta)
			if !ok {
				return nil, false
			}
			if texto == "" {
				muestra[j] = tipicos[j]
				break
			}
			v, err := strconv.ParseFloat(strings.ReplaceAll(texto, ",", "."), 64)
			if err != nil {
				fmt.Println("   Por favor escribe un número, por ejemplo 7.5")
				continue
			}
			if v < m.min || v > m.max {
				fmt.Printf("   Ese valor no es posible. Debe estar entre %g y %g.\n", m.min, m.max)
				continue
			}
			muestra[j] = v
			break
		}
	}
	return muestra, true
}

// mostrarResultado presenta la calidad estimada en lenguaje simple.
func mostrarResultado(modelo *forest.Forest, muestra []float64) int {
	votos := modelo.Votes(muestra)
	total := len(modelo.Trees)
	pred, max := 0, -1
	for c, v := range votos {
		if v > max {
			pred, max = c, v
		}
	}
	seguridad := float64(max) / float64(total) * 100
	nivel := "alta"
	if seguridad < 60 {
		nivel = "baja"
	} else if seguridad < 80 {
		nivel = "media"
	}

	fmt.Println()
	fmt.Println("==================== RESULTADO ====================")
	fmt.Printf("  Calidad del agua estimada:  %s\n", nivelesCalidad[pred].nombre)
	fmt.Printf("  Seguridad del resultado:    %s (%.0f%%)\n", nivel, seguridad)
	fmt.Println()
	fmt.Printf("  ¿Qué significa? %s\n", nivelesCalidad[pred].significado)
	if nivel != "alta" {
		segunda, votosSegunda := -1, 0
		for c, v := range votos {
			if c != pred && v > votosSegunda {
				segunda, votosSegunda = c, v
			}
		}
		if segunda >= 0 {
			fmt.Printf("  La muestra también se parece a una de calidad %s.\n", nivelesCalidad[segunda].nombre)
		}
	}
	fmt.Println()
	fmt.Println("  Este resultado es una estimación y no reemplaza un análisis")
	fmt.Println("  de laboratorio.")
	fmt.Println("===================================================")
	return pred
}

// valoresTipicos calcula la mediana de cada medición en los datos de
// entrenamiento, para ofrecerla como valor por defecto.
func valoresTipicos(X [][]float64) []float64 {
	n := len(X[0])
	res := make([]float64, n)
	col := make([]float64, len(X))
	for j := 0; j < n; j++ {
		for i, fila := range X {
			col[i] = fila[j]
		}
		sort.Float64s(col)
		res[j] = col[len(col)/2]
	}
	return res
}

func leerLinea(in *bufio.Reader, pregunta string) (string, bool) {
	fmt.Print(pregunta)
	linea, err := in.ReadString('\n')
	if err != nil && linea == "" {
		fmt.Println()
		return "", false
	}
	return strings.TrimSpace(linea), true
}

func unidadEntre(u string) string {
	if u == "" {
		return ""
	}
	return "en " + u + " "
}

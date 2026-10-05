# Informe de análisis técnico: HydroForest

## 1. Resumen ejecutivo

**Estado general.** HydroForest es un proyecto pequeño (~1 000 líneas de Go) y bien organizado. Implementa un Random Forest desde cero usando solo la librería estándar. El patrón concurrente de entrenamiento (cola de trabajos + pool de workers) es correcto y no se detectó ninguna condición de carrera, deadlock ni fuga de goroutines por lectura. Sus puntos débiles están en tres áreas:

- **Verificabilidad:** no hay pruebas automatizadas.
- **Validez y reproducibilidad del experimento:** el bosque concurrente no es determinista y la evaluación del modelo es débil.
- **Fidelidad del modelo Promela:** el modelo es más débil de lo que sugiere su nombre.

**GAPs identificados: 15.**

| Severidad | Cantidad | IDs |
|---|---|---|
| Alta | 3 | G-01, G-02, G-03 |
| Media | 5 | G-04, G-05, G-06, G-07, G-08 |
| Baja | 7 | G-09 a G-15 |

**Los 3 hallazgos más importantes**

1. **G-01.** `dataset.LoadCSV` descarta en silencio cualquier error de lectura del CSV (no solo `io.EOF`). Un CSV malformado produce un dataset truncado sin avisar.
2. **G-02.** No existe ningún archivo `_test.go` ni CI. La correctitud del árbol, de la concurrencia y de las métricas no se verifica automáticamente.
3. **G-03.** El entrenamiento concurrente no es reproducible: la semilla depende del worker, y el árbol que recibe cada worker depende del scheduler. Además, la "equivalencia funcional seq vs. concurrente" que se imprime en `main.go` no está garantizada por construcción.

## 2. Alcance y metodología

**Fuente analizada.** Clon superficial (`--depth 1`) de `https://github.com/ShoterBroXD/HydroForest`, rama principal, commit `251fd08`. El repositorio no se modificó.

**Archivos leídos completos**
- `go/cmd/hydroforest/{main,predict}.go`
- `go/internal/dataset/dataset.go`
- `go/internal/forest/{forest,model,sequential,concurrent,tree}.go`
- `go/internal/bench/bench.go`
- `go/go.mod`
- `promela/sync_model.pml`, `promela/sync_model_sin_mutex.pml`
- `go/results.csv`, `go/sweep.csv`
- `README.md`, `data/README.md`, `.gitignore`, `papers/referencias.md`

**Archivos leídos parcialmente**
- `scripts/graficos_speedup.py` (primeras ~45 líneas) y `scripts/graficos_sweep.py` (solo patrones de lectura y guardado de archivos). Son scripts de graficación y no se encontraron problemas relevantes.
- Celdas de código del notebook `notebooks/HydroForest_Limpieza_Dataset.ipynb` y sus salidas.

**Archivos no revisados (límites del análisis)**
- `docs/PC1/*.docx` y `docs/PC2/*.docx` (binarios) y las imágenes `docs/PC2/figuras/*.png`.
- El dataset real (304 MB) no está en el repositorio ni se descargó. Por eso **no se pudo confirmar ningún valor de accuracy**: el repositorio solo versiona tiempos.

**Limitaciones de entorno**
- En el equipo de análisis no están instalados `go` ni `spin`. No se compiló, no se ejecutó `go vet`, `go test -race` ni la verificación Promela. Todo es revisión estática.
- Los hallazgos que dependen de ejecución se marcan como **posible**, con la indicación de cómo confirmarlos.

**Criterios.** Calidad de código, seguridad, concurrencia, rendimiento, corrección del modelo de ML, pruebas y reproducibilidad. Las severidades se asignan por impacto real en un proyecto académico cuyo objetivo es demostrar speedup y corrección concurrente.

## 3. Fortalezas identificadas

- **Solo librería estándar:** cumple la restricción del curso. `encoding/csv`, `encoding/gob`, `sync`, `math/rand` y `sort` son suficientes.
- **Estructura clara:** `cmd/` y `internal/{dataset,forest,bench}` separan carga de datos, modelo y medición.
- **Patrón concurrente correcto** (`concurrent.go`):
  - Los trabajos se encolan en un canal con buffer del tamaño exacto y se cierra (`l.14-19`) antes de lanzar workers, así que no puede haber bloqueo de productor.
  - `wg.Add(1)` se ejecuta antes del `go` y `defer wg.Done()` está presente.
  - Cada goroutine tiene su propio `*rand.Rand`, porque `rand.Rand` no es seguro entre goroutines.
  - `X` e `y` se comparten solo en lectura.
  - El `append` está protegido por `sync.Mutex`.
- **Algoritmo de árbol correcto en lo esencial** (`tree.go`):
  - Gini con barrido sobre valores ordenados (O(n log n) por feature).
  - Umbral en el punto medio y salto de valores iguales.
  - Subconjunto aleatorio de √p features, bootstrap, condiciones de parada (profundidad, tamaño mínimo, pureza) y descarte de splits sin ganancia.
- **Evita la fuga evidente de la variable objetivo:** el notebook define `LEAKAGE_COL = "CCME_Values"` y el Go no la usa como feature.
- **Metodología de medición razonable:**
  - Media recortada con varias corridas.
  - Speedup y eficiencia.
  - Barrido fino de workers, volcado a CSV y graficación por script.
  - Coherencia verificable: con 1 worker el tiempo (247–252 s) coincide con el secuencial (249 s) en `results.csv` y `sweep.csv`.
- **Modo interactivo cuidado** (`predict.go`):
  - Valida rangos y acepta coma decimal.
  - Maneja EOF y valores por defecto (mediana).
  - Muestra el nivel de confianza por votos y un aviso de que no reemplaza un análisis de laboratorio.
- **Modelo Promela con control negativo:** incluye `sync_model_sin_mutex.pml`, que sirve para mostrar que la propiedad falla sin el mutex. Declara tres LTL (`exclusion_mutua`, `forest_completo`, `sin_deadlock`).
- **Higiene del repositorio y del dataset:**
  - Errores envueltos con `%w`, manejo de BOM y de CSV con columnas en distinto orden.
  - `.gitignore` adecuado (excluye CSV grandes y artefactos de Promela y gob).
  - `data/README.md` con cita, DOI y licencia CC BY 4.0.

## 4. GAPs identificados

| ID | Categoría | Ubicación | Descripción | Severidad |
|---|---|---|---|---|
| G-01 | Calidad / Corrección | `dataset/dataset.go:77-81` | `LoadCSV` termina el bucle ante *cualquier* error de `r.Read()`, no solo EOF; devuelve dataset truncado sin error | Alta |
| G-02 | Pruebas | Todo el repo | Sin `_test.go`, sin CI, sin evidencia de `go vet`/`-race` | Alta |
| G-03 | Concurrencia / Reproducibilidad | `forest/concurrent.go:22,27,36-41` | Semilla por worker + asignación no determinista de árboles + orden de inserción por finalización ⇒ bosque distinto en cada ejecución; no coincide con el secuencial | Alta |
| G-04 | Modelo ML | `main.go:98-104`, `forest.go:22-33` | Solo accuracy sobre un único split; sin matriz de confusión, F1/recall por clase ni intervalo | Media |
| G-05 | Modelo ML | `dataset.go:114-137`; notebook celdas 3 y 21 | Split aleatorio no agrupado/estratificado e imputación previa al split: posible fuga y accuracy optimista | Media |
| G-06 | Rendimiento | `tree.go:60-63,121-128,166-171`; `results.csv` | `sort.Slice` por nodo y feature, `[][]float64` y alocaciones por nodo; speedup se estanca en ~5× con 20 workers | Media |
| G-07 | Rendimiento / Metodología | `bench.go:12-27,189-199`; `sweep.csv` | Solo 3 muestras efectivas, sin dispersión; "equilibrio" frágil; seq y conc entrenan bosques distintos | Media |
| G-08 | Concurrencia / Modelo formal | `promela/*.pml` | El modelo no representa `Wait`, el canal ni la pérdida de actualización; LTL mal nombradas; posible error de sintaxis | Media |
| G-09 | Seguridad / Robustez | `forest/model.go`, `predict.go:70-82` | Modelo `.gob` sin metadatos ni validación; cualquier error de `Load` se trata como "no existe"; guardado no atómico | Baja |
| G-10 | Validación de entradas | `main.go:19-31,71-82`; `predict.go:148-156,209-220` | Flags sin validar (`runs<=0`, `trees<=0`); modo inválido se detecta tras cargar 2.8 M filas; `NaN` pasa la validación de rango | Baja |
| G-11 | Reproducibilidad / Docs | `README.md`, `data/README.md`, `main.go:57,65` | README sin instrucciones de Go; nombre de CSV inconsistente; semillas fijas y duplicadas; sin script de reproducción | Baja |
| G-12 | Manejo de errores | `main.go:124-141,201-232` | Errores de `csv.Writer` y de `Close` ignorados; se informa "guardado" aunque falle | Baja |
| G-13 | Calidad de código | varios | Duplicación (`PredictOne`/`Votes`, writers CSV, config de entrenamiento), E/S en librerías, sin comentarios en API exportada, shadowing de `max`/`real` | Baja |
| G-14 | Modelo ML | `tree.go:115-119` | Si las √p features sorteadas no tienen split válido, el nodo se vuelve hoja aunque otras features sí separen | Baja |
| G-15 | Rendimiento / Recursos | `predict.go:61-68,208-220` | El modo interactivo carga y ordena 8 columnas de ~2.3 M filas solo para medianas y una muestra de ejemplo | Baja |

---

### G-01: `LoadCSV` trunca el dataset en silencio ante errores de lectura (Alta)

**Descripción.** El bucle de lectura sale con `break` ante cualquier error, y la función retorna `nil` como error.

**Evidencia.**
```go
// dataset.go:77-81
for {
    row, err := r.Read()
    if err != nil {
        break
    }
```
`csv.Reader` devuelve errores que no son EOF:
- `ErrFieldCount`: `FieldsPerRecord` es 0 y se fija con la primera fila.
- `ErrQuote` y `ErrBareQuote`.
- Errores de E/S.

**Impacto.** Una fila corrupta en la posición 1 200 000 produce un dataset de 1.2 M filas. No hay error visible ni en `bench`, ni en `sweep`, ni en `predict`. Los resultados reportados quedarían calculados sobre datos parciales sin que nadie lo note. Es alto porque invalida silenciosamente todos los experimentos.

**Recomendación.**
```go
for line := 2; ; line++ {
    row, err := r.Read()
    if err == io.EOF { break }
    if err != nil { return nil, fmt.Errorf("línea %d: %w", line, err) }
    ...
}
```
Opcional: exponer el conteo de descartadas (`skipped`) como valor de retorno en lugar de imprimirlo, y rechazar `NaN`/`Inf` (ver G-10).

---

### G-02: Ausencia total de pruebas automatizadas y CI (Alta)

**Descripción.** El árbol de archivos versionados no contiene ningún `*_test.go`, workflow de CI ni `Makefile`. El README no menciona `go test`, `go vet` ni `-race`.

**Impacto.** Es un proyecto cuyo núcleo es la concurrencia y cuyo resultado principal es un speedup. Sin pruebas no hay forma de demostrar que:
- el bosque concurrente es equivalente al secuencial;
- el detector de carreras no marca nada;
- `TrimmedMean`, `bestSplit` o `MarkEquilibrium` hacen lo que dicen.

**Recomendación.**
1. Pruebas unitarias mínimas:
   - `giniImpurity` con casos conocidos.
   - `bestSplit` sobre un dataset de 6 puntos separable.
   - `TrimmedMean` (incluido `2*trim >= len`).
   - `Split` (tamaños y semilla).
   - `LoadCSV` con un CSV pequeño en `t.TempDir()`, incluida una fila con campos de menos.
2. Prueba de equivalencia, que depende de G-03:
```go
func TestSeqEqualsConc(t *testing.T) {
    X, y := synth(500, 8, 5)
    cfg := Config{NumTrees: 12, MaxDepth: 6, MinSamplesSplit: 5, NumClasses: 5, Seed: 1}
    a, b := TrainSequential(X, y, cfg), TrainConcurrent(X, y, cfg, 4)
    if !reflect.DeepEqual(a.Trees, b.Trees) { t.Fatal("bosques distintos") }
}
```
3. Documentar y ejecutar en cada cambio `go vet ./... && go test -race ./...`. Un workflow de GitHub Actions de 10 líneas basta.

---

### G-03: Entrenamiento concurrente no determinista y no equivalente al secuencial (Alta)

**Descripción.** El generador aleatorio de cada worker se siembra con `cfg.Seed + workerID + 1`. El árbol que procesa cada worker depende de qué goroutine lee antes del canal, es decir, del scheduler. El orden final de `trees` es el orden de finalización.

**Evidencia.**
```go
// concurrent.go:27, 36-41
rng := rand.New(rand.NewSource(cfg.Seed + int64(workerID) + 1))
for treeIdx := range jobs {
    sampleIdx := bootstrapSample(n, rng)
    tree := buildTree(X, y, sampleIdx, 0, cfg, rng)
    mu.Lock(); trees = append(trees, tree); mu.Unlock()
```
El secuencial usa un único flujo (`rand.NewSource(cfg.Seed)`, `sequential.go:11`). `treeIdx` solo se usa para el log.

**Impacto.**
- Dos ejecuciones con la misma semilla y el mismo número de workers pueden dar bosques y accuracies diferentes.
- Con distinto número de workers (`scalabilityCounts`, `FineSweep`), cada punto del barrido entrena un bosque distinto.
- Los mensajes `Accuracy ... equivalencia funcional seq vs. concurrente` (`main.go:98-104`) comparan bosques estadísticamente similares, no equivalentes. La afirmación no está garantizada por construcción.

**Recomendación.** Sembrar por **árbol**, no por worker, y escribir en el índice del árbol. Esto elimina también la necesidad del mutex (ver sección 5):
```go
trees := make([]*Node, cfg.NumTrees)
// en el worker:
for i := range jobs {
    rng := rand.New(rand.NewSource(cfg.Seed + int64(i)))
    trees[i] = buildTree(X, y, bootstrapSample(n, rng), 0, cfg, rng)
}
```
Usar el mismo esquema en `TrainSequential`. Así ambos producen exactamente el mismo bosque y la prueba de G-02 pasa con `DeepEqual`.

---

### G-04: Métricas de evaluación insuficientes (Media)

**Descripción.** Solo se calcula accuracy global (`Forest.Accuracy`), en un único split 80/20 con semilla 7. No hay matriz de confusión, precision/recall/F1 por clase, F1 macro, ni repetición del split. El repositorio tampoco versiona ningún resultado de accuracy (solo tiempos).

**Impacto.** Con 5 clases ordinales probablemente desbalanceadas (el notebook grafica la distribución pero la salida no se incluye), la accuracy global puede ocultar un mal desempeño en clases minoritarias. Esto debilita cualquier conclusión sobre "calidad del modelo". Para confirmar el desbalance hay que ejecutar el notebook.

**Recomendación.** Añadir una función `Confusion(X, y)` que devuelva una matriz `[5][5]int`, y derivar de ella precision, recall y F1 por clase más F1 macro. Imprimir y volcar a CSV junto con los tiempos. Es un cambio de ~30 líneas con la estándar.

---

### G-05: Posible fuga de información y split optimista (Media; *posible*)

**Descripción.** El split es una permutación aleatoria global (`dataset.Split`), sin estratificar ni agrupar por estación (`Area`). El notebook muestra que la misma `Area` tiene una fila por año (celda 5: `SE649035-145565`, 1974, 1975, 1976…), y la imputación por mediana de `Country` (celda 21) se calcula sobre todo el dataset antes de partir.

**Evidencia.** `dataset.go:114-137` y notebook, celdas 3, 5 y 21. La columna `Area` y `Date` se limpian pero no llegan al CSV que lee Go como feature (solo 8 features).

**Impacto.** Mediciones consecutivas de una misma estación son muy parecidas y comparten clase. Un split aleatorio puede poner vecinas en train y test, lo que infla la accuracy respecto a generalizar a estaciones nuevas. Además, el índice CCME se calcula a partir de estos mismos parámetros, así que el problema es en gran medida reconstruir una fórmula (esto es una inferencia a confirmar: requiere cotejar la metodología del dataset). El impacto real solo se puede cuantificar ejecutando ambos tipos de split.

**Cómo confirmarlo.** Entrenar con split aleatorio y con split agrupado por `Area` (o temporal por `Date`), y comparar accuracy y F1. Si difieren de forma apreciable, hay fuga.

**Recomendación.** Exportar `Area` y `Date` (o un id de grupo) en el CSV limpio y agregar `SplitGrouped(d, groups, trainFrac, seed)`. Calcular las medianas de imputación solo con el train, o documentar la limitación como amenaza a la validez en el informe.

---

### G-06: Cuello de botella en `bestSplit` y escalamiento limitado a ~5× (Media; la causa es *posible*)

**Descripción.** En cada nodo y para cada feature candidata se copia `indices` a `order` y se ordena con `sort.Slice`, cuyo comparador es una closure que accede a `X[order[a]][f]` (doble indirección: `[][]float64`, filas separadas en el heap). Además `leftIdx`/`rightIdx` crecen con `append` sin capacidad previa, y se asigna un `order` y un `bootstrapSample` de tamaño n por nodo y por árbol.

**Evidencia.**
```go
// tree.go:60-63
order := make([]int, n)
for _, f := range candidateFeatures {
    copy(order, indices)
    sort.Slice(order, func(a, b int) bool { return X[order[a]][f] < X[order[b]][f] })
```
`results.csv`: el speedup con 20 workers es 5.25–5.42× (eficiencia 0.26–0.27). Con 8 workers ya está en 4.46×. El barrido (`sweep.csv`) pasa de 4.46× (7 workers) a 4.7× (10–15 workers).

**Impacto.** El algoritmo se desarrolla casi todo en memoria con acceso disperso y mucha alocación (presión de GC compartida entre goroutines). Esto es un patrón típico de saturación de ancho de banda de memoria, pero con solo estos datos **no se puede afirmar la causa**. Otras explicaciones posibles: 20 CPUs lógicos con SMT/E-cores (no verificado), granularidad de 60 árboles entre 20 workers, o frecuencia térmica.

**Cómo confirmarlo.** Perfilar con `runtime/pprof` o `go test -bench -cpuprofile`, y comparar con `GOGC` alto. Comparar el speedup con el número de núcleos *físicos*.

**Recomendación.**
- Reutilizar buffers por worker (`order`, `leftCounts`, `rightCounts`) en lugar de alocar por nodo.
- Usar `slices.SortFunc` (estándar desde Go 1.21; el proyecto usa 1.24) o, mejor, preordenar cada feature una sola vez por árbol (índices ordenados) y particionar en lugar de reordenar en cada nodo.
- Almacenar los datos por columnas (`[][]float64` por feature) para mejorar la localidad de caché.

Estos cambios no alteran el modelo y mejoran el tiempo secuencial y concurrente. Es una optimización de alcance medio y se debe reportar el efecto sobre el speedup (que puede *bajar* aunque el tiempo absoluto mejore, lo que debe explicarse en el informe).

---

### G-07: Metodología de benchmark frágil (Media)

**Descripción.**
1. Con `-runs 5 -trim 1` (valores por defecto) la media recortada usa 3 valores. No se reporta desviación ni intervalo, y no hay calentamiento.
2. `MarkEquilibrium` marca el primer `w` cuya *siguiente* ganancia marginal sea < 5%. Con datos ruidosos eso es frágil.
3. `TimeRuns` mide con distintos bosques (G-03). Se mide tiempo de bosques que no son idénticos.
4. En modo `bench` se vuelven a entrenar ambos bosques solo para calcular accuracy (`main.go:99-100`), duplicando ~5 min de cómputo.

**Evidencia.** `sweep.csv`: la fila de 5 workers queda marcada `equilibrio=1` (ganancia siguiente 1.12%). Sin embargo, pasar a 7 workers aún da 10.19% y de 19 a 20 workers da 17.44% (62.1 → 55.8 → … → 46.4 s). La serie no es monótona (8→9 workers *empeora* 5.66%), lo que indica ruido o efectos de plataforma y no un "punto de equilibrio" estable.

**Impacto.** La conclusión "el equilibrio está en 5 workers" no es robusta con estos datos. Si el informe la presenta como resultado, es discutible.

**Recomendación.** Más corridas (≥10), reportar mediana y desviación estándar o rango, y definir el equilibrio con una regla más robusta (por ejemplo, el menor `w` cuya mediana esté dentro de un x% del mejor tiempo observado). Entrenar una sola vez por versión y reutilizar ese bosque para el accuracy.

---

### G-08: Fidelidad limitada del modelo Promela (Media)

**Descripción.** Se verifican propiedades sobre un modelo que abstrae la lógica de forma demasiado favorable.

**Evidencia.**
- **`trees_built++` es un único paso atómico en Promela** (`sync_model.pml:37`). Por tanto la pérdida de actualización de un `append` no protegido (leer-modificar-escribir) *no puede modelarse*. En la versión sin mutex, lo único que falla es el `assert(in_critical == 1)`, una variable de instrumentación, no la integridad de los datos. La propiedad `forest_completo` se cumpliría incluso sin mutex.
- **Nombres de propiedades engañosos.** `sin_deadlock` es `<> [] (wg_counter == 0)`: comprueba terminación de los workers, no ausencia de deadlock. El `wg.Wait()` del hilo principal **no existe** en el modelo, así que lo realmente delicado (que `Wait` retorne) no se verifica.
- **Equivalencia con el canal.** `jobs_remaining` modela la cola como un contador atómico, no como un canal con cierre. Es una abstracción razonable, pero debe documentarse.
- **Posible error de sintaxis (a confirmar con `spin -a`).** En `sync_model_sin_mutex.pml:33` hay `skip` sin `;` antes de `in_critical++;`, y en `sync_model.pml:34` falta un `;` tras el bloque `atomic`. Spin acepta la omisión tras `}`, pero **no tras `skip`**. Hay que ejecutar `spin -a` para confirmarlo.
- **Sin instrucciones de verificación.** Las propiedades `<>[]` requieren equidad débil (`pan -a -f`); sin ella pueden dar falsos contraejemplos. Tampoco se documenta cómo ejecutar cada propiedad ni qué resultado se espera.

**Impacto.** El modelo respalda la afirmación "no hay race ni deadlock" con menos fuerza de la que parece. Con tres workers y seis trabajos el espacio de estados es mínimo, así que sería barato modelar mejor.

**Recomendación.**
- Modelar la inserción como lectura y escritura separadas para que el contraejemplo sea real:
```promela
int tmp; tmp = trees_built; trees_built = tmp + 1;   /* sin mutex: lost update */
```
- Añadir un proceso `Main` que espere `wg_counter == 0` y luego afirme `trees_built == TOTAL_JOBS`. Renombrar `sin_deadlock` a `terminacion` y verificar ausencia de deadlock con `pan -E` o comprobando que no haya estados finales inválidos.
- Documentar los comandos exactos y el resultado esperado.

---

### G-09: Modelo `.gob` sin metadatos ni validación (Baja)

**Descripción.**
1. `Forest.Load` decodifica sin validar la estructura: un archivo alterado o de otro dataset provoca un pánico por índice fuera de rango (`x[node.FeatureIdx]`, `predictTree`), `nil` en `Left`/`Right` de un nodo no hoja, o `nivelesCalidad[pred]` con `pred` fuera de 0..4.
2. El modelo no guarda semilla, hiperparámetros, versión de features ni hash del CSV. `predict` reutiliza un `.gob` anterior aunque el dataset o los `FeatureNames` hayan cambiado.
3. Cualquier error de `Load` (incluido un archivo corrupto) se interpreta como "primera vez" y se reentrena (`predict.go:70-73`).
4. `Save` escribe directo sobre el destino y no comprueba el error de `Close` (`model.go:21-29`).

**Impacto.** Bajo para un uso local y académico: `gob` es memoria-segura y no ejecuta código. El riesgo es de robustez (pánicos, modelo obsoleto) y de DoS con un archivo adversarial, no de ejecución remota.

**Recomendación.** Guardar un encabezado `{Version, FeatureNames, NumClasses, Cfg}` y validar al cargar. Escribir a un archivo temporal y `os.Rename`. Distinguir `os.IsNotExist` de errores de decodificación.

---

### G-10: Validación de entradas incompleta (Baja)

**Descripción.**
- `-runs <= 0`: `TimeRuns` devuelve vacío y `TrimmedMean` retorna 0, así que el speedup es `+Inf`/`NaN`. `-trees <= 0` produce un bosque vacío que "predice" siempre la clase 0 sin error. `-workers` negativo en `demo` se corrige, pero en `TrainConcurrent` un `numWorkers == 0` devuelve un bosque vacío en silencio.
- El modo inválido se detecta recién en `main.go:71-82`, **después** de cargar y particionar 2.8 M filas.
- `strconv.ParseFloat` acepta `"NaN"` y `"Inf"`. En `predict.go:148-156` la comparación `v < m.min || v > m.max` es falsa para `NaN`, así que **`NaN` pasa la validación** (`Inf` sí es rechazada por rango). En `LoadCSV` un `NaN` en el CSV entra a los datos y rompe el orden del `sort`.
- `valoresTipicos` hace `X[0]` y falla con pánico si el dataset quedó vacío.

**Impacto.** Bajo: los usuarios son los propios autores o estudiantes, pero el caso `-runs 0` produce números sin sentido sin avisar.

**Recomendación.**
```go
if math.IsNaN(v) || math.IsInf(v, 0) || v < m.min || v > m.max { ... }
```
Validar flags (`runs>=1`, `trees>=1`, `0<=trim`, `2*trim<runs`) y el `mode` antes de cargar datos; hacer que `TrainConcurrent` rechace `numWorkers < 1` o lo ajuste a 1.

---

### G-11: Reproducibilidad y documentación del experimento (Baja)

**Descripción.**
- `README.md` solo explica cómo limpiar el dataset con el notebook. No dice cómo compilar o ejecutar el programa Go (`go run ./cmd/hydroforest -data ...`), ni qué hace cada `-mode`, ni requisitos (Go ≥ 1.24.7, RAM, tiempo estimado ≈ 5 min por corrida de bench).
- Inconsistencia de nombres: `data/README.md` pide guardar el CSV como `water_quality.csv`; el Go espera columnas ya normalizadas (`pH_ph_units`, …) que solo produce el notebook (`water_quality_clean.csv`). Un usuario que pase el CSV original obtiene "columna requerida no encontrada".
- Las semillas de split (7) y de modelo (42) están fijas y duplicadas en `main.go:57,65` y `predict.go:67,73` sin flag. Si una cambia, `predict` evalúa la muestra "real" sobre un test distinto al del entrenamiento (fuga en la demo).
- `.gitignore` excluye `*.csv`, correcto, pero no hay un script único (Makefile o `run.sh`) que encadene limpieza → bench → gráficos. El notebook depende de pandas/matplotlib, sin `requirements.txt`.
- Los resultados (`results.csv`, `sweep.csv`) no registran hardware, versión de Go, `GOMAXPROCS` ni semilla.

**Impacto.** Un revisor externo necesita leer el código para ejecutar el experimento. Es un problema de reproducibilidad, no de corrección.

**Recomendación.** Ampliar el README con comandos, requisitos y tiempos. Exponer `-seed`/`-split-seed` como flags con constantes compartidas, y añadir una cabecera de metadatos (`# go1.24.7 cpu=20 seed=42`) al CSV de resultados. Incluir `requirements.txt` para el notebook y los scripts.

---

### G-12: Errores de escritura ignorados (Baja)

**Descripción.** `writeResultsCSV` y `writeSweepCSV` ignoran el valor de retorno de `w.Write`, no consultan `w.Error()` tras `Flush` y no verifican el error de `f.Close()` (`main.go:124-141`, `201-232`).

**Impacto.** Con disco lleno o permisos incorrectos se imprime "Tabla de resultados guardada" aunque el archivo esté incompleto, y los scripts de graficación leerían datos truncados.

**Recomendación.**
```go
w.Flush()
if err := w.Error(); err != nil { return err }
return f.Close()
```
sin `defer f.Close()` para ese camino (o usar un cierre con retorno nombrado).

---

### G-13: Calidad de código y mantenibilidad (Baja)

**Descripción.**
- Duplicación: `PredictOne` y `Votes` repiten el mismo conteo (`forest.go:8-20` vs `model.go:9-16`); `mostrarResultado` reimplementa el argmax; `writeResultsCSV`/`writeSweepCSV` comparten la estructura; los hiperparámetros del modo `predict` se repiten literalmente (`predict.go:73`) en vez de venir de `Config` por defecto.
- Efectos de E/S dentro de bibliotecas: `LoadCSV` imprime con `fmt.Printf`; `forest` imprime en modo `Verbose`; `bench` mezcla cálculo e impresión. Dificulta las pruebas.
- Falta de comentarios `godoc` en casi todas las funciones y tipos exportados (`Dataset`, `LoadCSV`, `Split`, `TrainSequential`, `TrainConcurrent`, `Config`, `Node`).
- `predict.go` (237 líneas) mezcla UI, textos y lógica en `package main`. Los identificadores `max` y `real` ocultan builtins.
- Algunos mensajes de usuario están en español y los identificadores del dominio en español e inglés mezclados, sin una convención.

**Impacto.** Bajo; es legible y el tamaño es manejable. Afecta la mantenibilidad futura y la testabilidad.

**Recomendación.** Implementar `PredictOne` sobre `Votes`; mover `argmax` a `forest`; pasar un `io.Writer` o logger opcional a las funciones que imprimen.

---

### G-14: Hoja prematura cuando las features sorteadas no separan (Baja)

**Descripción.** `randomFeatureSubset` devuelve √8 = 2 features. Si en un nodo ninguna de las 2 produce un split válido, el nodo se convierte en hoja (`tree.go:115-119`), aunque alguna de las otras 6 sí podría separarlo. Las implementaciones de referencia (p. ej. scikit-learn) siguen buscando hasta encontrar al menos una feature utilizable.

**Impacto.** Árboles más superficiales de lo necesario y posible pérdida de accuracy, sobre todo en nodos grandes. No afecta la corrección formal (sigue siendo un clasificador válido) y no se puede cuantificar sin ejecutar.

**Recomendación.** Si no hay split, probar las features restantes en un segundo intento.

---

### G-15: Arranque costoso del modo interactivo (Baja)

**Descripción.** `runPredict` carga el CSV completo y calcula las medianas con `sort.Float64s` sobre 8 columnas de ~2.26 M valores, solo para ofrecer valores por defecto y una muestra de ejemplo (`predict.go:61-68,208-220`). Aun cuando el modelo ya está guardado, se necesita el dataset de 304 MB (más ~300 MB de RAM en `[][]float64`).

**Impacto.** El usuario final espera un arranque rápido; aquí requiere el dataset en disco y varios segundos o decenas de segundos de carga. No es un error, pero va contra la idea de un modo "para usuarios finales".

**Recomendación.** Guardar en el `.gob` las medianas (y unas decenas de muestras de ejemplo) junto con el modelo. Así `predict` no necesita el CSV una vez entrenado.

---

## 5. Análisis de concurrencia

### 5.1 Diseño implementado

`TrainConcurrent` implementa un **pool de workers con cola de trabajos**:

1. Un canal con buffer `NumTrees` se llena con los índices de árbol y se cierra antes de iniciar los workers.
2. Se lanzan `numWorkers` goroutines, cada una con su propio generador aleatorio. Cada goroutine consume del canal (`for treeIdx := range jobs`), construye el árbol y lo agrega a un slice compartido bajo `sync.Mutex`.
3. El hilo principal espera con `sync.WaitGroup`.

El patrón es adecuado: el trabajo está naturalmente particionado por árbol (embarrassingly parallel), y la cola dinámica balancea la carga mejor que una partición estática, dado que los árboles tienen tamaños distintos.

### 5.2 Evaluación de seguridad concurrente (por lectura)

| Aspecto | Resultado | Evidencia |
|---|---|---|
| Carreras de datos | **No se detectan** | `X`, `y` y `cfg` solo se leen; `rng` es local a cada goroutine; `trees` solo se toca bajo `mu` |
| Deadlock | **No** | El canal está lleno y cerrado antes de los workers; `Lock`/`Unlock` sin llamadas intermedias bloqueantes |
| Fuga de goroutines | **No** | `range jobs` termina al agotarse el canal; `defer wg.Done()` |
| `wg.Add` | Correcto | Se ejecuta antes de `go` en el hilo principal |
| Pánico en worker | Termina todo el proceso | No hay `recover`; aceptable aquí, pero conviene saberlo |
| Salida por consola | Segura | `fmt.Printf` serializa por llamada; las líneas podrían intercalarse entre workers pero no se corrompen |

Esta conclusión es **por revisión estática**. Falta confirmarla con `go test -race` (ver G-02), porque Go no está instalado en el entorno de análisis.

### 5.3 Observaciones

- **El mutex es innecesario.** Cada árbol tiene un índice propio, así que escribir en `trees[i]` desde su worker no tiene conflicto: no hay dos goroutines que escriban en la misma posición. Con eso se elimina el mutex, se arregla el orden del bosque y se logra el determinismo (G-03). El mutex actual no es incorrecto; es una sincronización que se podría evitar, y su existencia es la que se está modelando en Promela.
- **Granularidad.** Con 60 árboles y 20 workers, cada worker procesa ~3 árboles. Un árbol grande al final puede dejar workers ociosos (efecto cola). Esto es una hipótesis (*posible*) del comportamiento no monótono en `sweep.csv`; se confirmaría con los registros del modo `demo`, que ya imprimen inicio y fin por árbol y por worker.
- **Contención oculta.** Los workers comparten el *heap* y el GC (G-06). Esto no es una carrera, pero limita el escalamiento.

### 5.4 Correspondencia Promela ↔ Go

| Elemento Go | Elemento Promela | Correspondencia |
|---|---|---|
| Canal `jobs` pre-llenado y cerrado | `jobs_remaining` con decremento atómico | Abstracción razonable. No modela cierre ni semántica de canal |
| `numWorkers` goroutines | `active [NUM_WORKERS] proctype Worker` | Correcta (3 frente a 20: suficiente para el interleaving) |
| `sync.Mutex` | `bit mutex` con `atomic { mutex == 0 -> mutex = 1 }` | Correcta |
| `append(trees, tree)` | `trees_built++` | **Débil**: el incremento es atómico en Promela, no modela la pérdida de actualización |
| `wg.Add(1)` por worker | `wg_counter = NUM_WORKERS` | Correcta |
| `defer wg.Done()` | `wg_counter--` | Correcta; no es atómica con el cierre del bucle, pero es un único paso |
| `wg.Wait()` | **Ausente** | No hay proceso `Main`; la propiedad más relevante (que `Wait` retorne) no se verifica |
| Propiedad "sin race" | `in_critical <= 1` | Es una variable de instrumentación; válida como oráculo de exclusión mutua |

**Conclusión de la correspondencia.** El modelo captura bien la estructura (productor/consumidor, exclusión mutua, contador de espera) y el control negativo sin mutex es una buena idea. Su debilidad está en que no puede demostrar la corrupción de datos que un mutex evita, y en que no modela el `Wait`. Con las correcciones de G-08 quedaría alineado con el código real. Además, si se adopta la recomendación de G-03 (escritura por índice, sin mutex), hay que **actualizar el modelo** para que siga reflejando la implementación: ya no habría sección crítica sobre `trees`, y la propiedad a verificar pasaría a ser que cada índice se escribe exactamente una vez.

## 6. Plan de mejoras priorizado

| # | Acción | GAPs | Prioridad | Esfuerzo |
|---|---|---|---|---|
| 1 | Corregir `LoadCSV` (distinguir `io.EOF`, devolver error con número de línea) y añadir su prueba | G-01 | Alta | Bajo |
| 2 | Sembrar por árbol y escribir en `trees[i]`; igualar `TrainSequential`; quitar el mutex | G-03 | Alta | Bajo |
| 3 | Crear pruebas unitarias y de equivalencia seq = conc; ejecutar `go vet` y `go test -race`; añadir CI mínimo | G-02 | Alta | Medio |
| 4 | Añadir matriz de confusión, F1 por clase y F1 macro; guardar resultados de accuracy en `results.csv` | G-04 | Media | Bajo |
| 5 | Corregir y ampliar el modelo Promela (`Wait`, lost update, renombrar LTL, documentar `spin`/`pan`) y verificar con Spin | G-08 | Media | Medio |
| 6 | Mejorar el benchmark: más corridas, dispersión, regla de equilibrio robusta, un solo entrenamiento por versión | G-07 | Media | Bajo |
| 7 | Comparar split aleatorio frente a agrupado por `Area`/temporal; documentar la amenaza a la validez | G-05 | Media | Medio |
| 8 | Perfilar con `pprof`; reutilizar buffers y reducir el costo de ordenar; medir el efecto en el speedup | G-06 | Media | Alto |
| 9 | Ampliar el README (ejecución Go, requisitos, tiempos, nombres de archivos), exponer flags de semilla, `requirements.txt`, script de reproducción | G-11 | Baja | Bajo |
| 10 | Validar flags y entradas (`NaN`, `runs`, `trees`, `workers`, `mode` antes de cargar) | G-10 | Baja | Bajo |
| 11 | Hacer robusto el modelo guardado (encabezado, validación, escritura atómica) y guardar medianas en él | G-09, G-15 | Baja | Medio |
| 12 | Comprobar errores de escritura CSV | G-12 | Baja | Bajo |
| 13 | Refactor menor (deduplicar, `godoc`, sacar E/S de las bibliotecas) | G-13 | Baja | Medio |
| 14 | Reintentar con features restantes al no hallar split | G-14 | Baja | Bajo |

**Orden sugerido para una entrega próxima:** 1 → 2 → 3 → 4 → 5. Son baratos, resuelven las tres severidades Altas y refuerzan directamente los argumentos de corrección concurrente del proyecto.

## 7. Conclusión

HydroForest cumple lo central del curso: un Random Forest propio, solo con la librería estándar de Go, con un diseño concurrente simple y correcto por inspección, un módulo de medición con speedup y escalabilidad, y un modelo Promela de la sincronización. La arquitectura del código es limpia y el modo interactivo está bien pensado.

Lo que le falta es **evidencia verificable**:
- no hay pruebas automatizadas;
- el entrenamiento concurrente no es reproducible, por lo que no se puede afirmar equivalencia con el secuencial;
- las métricas de calidad del modelo son mínimas;
- el modelo Promela no demuestra la propiedad que más importa (ausencia de pérdida de datos y retorno de `Wait`);
- un error silencioso en la lectura del CSV podría invalidar los resultados sin avisar.

Ninguno de estos puntos exige rediseñar el proyecto. Las acciones 1 a 6 del plan son de esfuerzo bajo o medio y bastarían para pasar de "funciona y es razonable" a "demostrablemente correcto y reproducible".

Por último, se recuerda que este informe es una revisión estática: no se compiló ni se ejecutó el código, ni se corrió Spin, ni se usó el dataset real. Los puntos marcados como *posible* (G-05, G-06, G-08 en la sintaxis, y las hipótesis de la sección 5.3) requieren las verificaciones indicadas en cada GAP.

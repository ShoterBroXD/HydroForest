# Referencias bibliográficas

Papers investigados por cada integrante del grupo, centrados en modelos de árboles de decisión / Random Forest combinados con técnicas de programación concurrente, paralela o distribuida en los últimos 5 años.

## Diego Fernando Meléndez Marín (U202220345)

1. **A Massively Parallel SMC Sampler for Decision Trees**
   Drousiotis, E.; Varsi, A.; Phillips, A. M.; Maskell, S.; Spirakis, P. G. (2025). _Algorithms_, 18(1), 14.
   Paralelización del paso de remuestreo (redistribution) en árboles de decisión bayesianos, usando OpenMP (memoria compartida) y MPI (memoria distribuida).
   Link: https://www.mdpi.com/1999-4893/18/1/14

2. **A Fast Parallel Random Forest Algorithm Based on Spark**
   Yin, L.; Chen, K.; Jiang, Z.; Xu, X. (2023). _Applied Sciences_, 13(10), 6121.
   Nuevo coeficiente de Gini y tabla FSI para acelerar el entrenamiento distribuido de Random Forest sobre Apache Spark.
   Link: https://www.mdpi.com/2076-3417/13/10/6121

3. **A Parallel Approach to Enhance the Performance of Supervised Machine Learning Realized in a Multicore Environment**
   Ghimire, A.; Amsaad, F. (2024). _Machine Learning and Knowledge Extraction_, 6(3), 1840-1856.
   Paralelización de Random Forest por bootstrap sampling, asignando un árbol independiente a cada núcleo del procesador.
   Link: https://www.mdpi.com/2504-4990/6/3/90

---

## Angel Gabriel Díaz Chavez (U20221C424)

1. **Multi-GPU approach to global induction of classification trees for large-scale data mining**
   Jurczuk, K., Czajkowski, M., & Kretowski, M. (2021). _Applied Intelligence_, 51(8), 5683–5700.
   Propone repartir el dataset entre varias GPUs para acelerar la inducción evolutiva de árboles de clasificación sobre datasets de gran escala, sincronizando en la CPU los resultados parciales calculados en paralelo por cada GPU.
   Link: https://doi.org/10.1007/s10489-020-01952-5

2. **GPU-based acceleration of evolutionary induction of model trees**
   Jurczuk, K., Czajkowski, M., & Kretowski, M. (2022). _Applied Soft Computing_, 119, 108503.
   Diseña seis procedimientos de aceleración por GPU para el entrenamiento evolutivo de Model Trees, paralelizando tanto la evaluación de instancias como el ajuste de regresiones lineales en las hojas.
   Link: https://doi.org/10.1016/j.asoc.2022.108503

3. **From distributed machine to distributed deep learning: a comprehensive survey**
   Dehghani, M., & Yazdanparast, Z. (2023). _Journal of Big Data_, 10(1), 158.
   Revisa y clasifica las principales estrategias de paralelización y distribución en machine learning (clasificación, clustering, deep learning y aprendizaje por refuerzo), destacando el paralelismo de datos como patrón base aplicable a nuestro enfoque con goroutines.
   Link: https://doi.org/10.1186/s40537-023-00829-x

---

## Diego Tomás Villafuerte Ramirez (U202010546)

1. **SPMD-Based Neural Network Simulation with Golang**
   Kalwarowskyj, D.; Schikuta, E. (2023). _Computational Science - ICCS 2023, Lecture Notes in Computer Science_, 14075, 563-570.
   Entrenamiento data-paralelo bajo el modelo SPMD en Go: el dataset se divide entre redes hijas que entrenan en goroutines sincronizadas con sync.WaitGroup y sync.Mutex, y luego se promedian sus pesos.
   Link: https://doi.org/10.1007/978-3-031-36024-4_43

2. **MLaaS4HEP: Machine Learning as a Service for HEP**
   Kuznetsov, V.; Giommi, L.; Bonacorsi, D. (2021). _Computing and Software for Big Science_, 5(1), 17.
   Pipeline de tres capas que entrena sobre 28.5 millones de registros leídos por bloques sin cargarlos completos en memoria, con la capa de inferencia implementada en Go por su soporte nativo de goroutines y canales.
   Link: https://doi.org/10.1007/s41781-021-00061-3

3. **Sustainable Image Processing for Digital News Platforms: Evaluating Go Concurrency Models for Efficient Media Workloads**
   Avicenna, H. R.; Widiyanto, A.; Hasani, R. A. (2026). _E3S Web of Conferences_, 706, 03008.
   Comparación empírica en Go entre concurrencia no acotada (una goroutine por tarea) y un worker pool acotado con canales con buffer: el pool alcanza 3.74x de aceleración mientras la versión sin límite falla por falta de memoria.
   Link: https://doi.org/10.1051/e3sconf/202670603008

# HydroForest

Clasificación concurrente de calidad de agua con Random Forest en Go, alineado con el ODS 6 (Agua Limpia y Saneamiento).

Proyecto del curso **Programación Concurrente y Distribuida**, UPC, ciclo 2026-2.

## Sobre el proyecto

El objetivo es implementar un modelo de clasificación de calidad de agua sobre un dataset de más de 2.8 millones de registros, comparando una implementación secuencial contra una versión concurrente en Go para medir el speedup y la escalabilidad obtenidos.

## Dataset

**A Comprehensive Surface Water Quality Monitoring Dataset (1940-2023)**, publicado en Figshare (2.82M de registros, 14 variables, incluye la clasificación `CCME_WQI` usada como variable objetivo).

Enlace de descarga: https://figshare.com/articles/dataset/A_Comprehensive_Surface_Water_Quality_Monitoring_Dataset_1940-2023_2_82Million_Record_Resource_for_Empirical_and_ML-Based_Research/27800394

El CSV original no se encuentra en este repositorio debido al límite de tamaño permitido para subir. Ver `data/README.md` para instrucciones de descarga.

## Reproducir la limpieza del dataset

1. Descargar el CSV desde el link de Figshare y colocarlo en la misma carpeta que el notebook.
2. Abrir `notebooks/HydroForest_Limpieza_Dataset.ipynb` en Jupyter o Google Colab.
3. Ajustar `INPUT_PATH` si el nombre del archivo es distinto.
4. Ejecutar todas las celdas.

El notebook genera `water_quality_clean.csv`, además de las gráficas de nulos y distribución de clases usadas en el informe.

## Documentación

- Informe completo (TB1): [`docs/TB1/CC65-TB1-202620.docx`](docs/TB1/CC65-TB1-202620.docx)
- Papers investigados: [`papers/referencias.md`](papers/referencias.md)

## Integrantes

| Nombre | Código |
|---|---|
| Diego Fernando Meléndez Marín | U202220345 |
| Angel Gabriel Díaz Chavez | U20221C424 |
| Diego Tomás Villafuerte Ramirez | U202010546 |
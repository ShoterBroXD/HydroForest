"""
Genera los gráficos del barrido fino de workers (HydroForest, CC65) a partir
de go/sweep.csv, generado con:

    go run ./cmd/hydroforest -data ../data/water_quality_clean.csv -mode sweep ...

Uso (desde la raíz del repo):
    python scripts/graficos_sweep.py
    python scripts/graficos_sweep.py --csv go/sweep.csv --out docs/PC2/figuras --threshold 5

Requiere: matplotlib  (pip install matplotlib)
"""
import argparse
import csv
import os

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt

TEAL = "#008C9E"
GRAY = "#A3B1B4"
INK = "#1F2A2D"
MUTED = "#5C6B6E"
GRID = "#E6EAEB"

plt.rcParams.update({
    "font.family": "DejaVu Sans", "font.size": 11,
    "axes.edgecolor": GRID, "axes.labelcolor": MUTED,
    "xtick.color": MUTED, "ytick.color": MUTED,
    "axes.spines.top": False, "axes.spines.right": False,
    "figure.facecolor": "white", "axes.facecolor": "white",
})


def leer_sweep(path):
    filas = []
    with open(path, newline="", encoding="utf-8") as f:
        for row in csv.DictReader(f):
            filas.append({
                "workers": int(row["workers"]),
                "tiempo": float(row["media_recortada_s"]),
                "speedup": float(row["speedup"]),
                "eficiencia": float(row["eficiencia"]),
                "ganancia": float(row["ganancia_marginal_pct"]) if row["ganancia_marginal_pct"] else None,
                "equilibrio": row["equilibrio"] == "1",
                "baseline": float(row["baseline_secuencial_s"]),
            })
    if not filas:
        raise SystemExit(f"{path} está vacío")
    filas.sort(key=lambda r: r["workers"])
    return filas


def estilo(ax):
    ax.grid(axis="y", color=GRID, linewidth=1)
    ax.set_axisbelow(True)
    ax.spines["left"].set_visible(False)
    ax.tick_params(length=0)


def titulo(ax, texto, subtitulo):
    ax.set_title(texto, loc="left", color=INK, fontsize=13, fontweight="bold", pad=22)
    ax.text(0, 1.04, subtitulo, transform=ax.transAxes, color=MUTED, fontsize=10)


def grafico_tiempo(filas, out):
    workers = [r["workers"] for r in filas]
    tiempos = [r["tiempo"] for r in filas]
    eq = next((r for r in filas if r["equilibrio"]), None)

    fig, ax = plt.subplots(figsize=(7.5, 4.3), dpi=200)
    ax.plot(workers, tiempos, color=TEAL, linewidth=2, marker="o", markersize=6,
            markeredgecolor="white", markeredgewidth=1.5, zorder=3)
    if eq:
        ax.axvline(eq["workers"], color=GRAY, linewidth=1.5, linestyle=(0, (5, 4)), zorder=1)
        ax.plot([eq["workers"]], [eq["tiempo"]], marker="o", markersize=11, color=INK,
                markeredgecolor="white", markeredgewidth=2, zorder=4)
        ax.annotate(f"Punto de equilibrio\n{eq['workers']} workers, {eq['tiempo']:.1f} s",
                    xy=(eq["workers"], eq["tiempo"]), xytext=(12, 28), textcoords="offset points",
                    color=INK, fontsize=10, fontweight="bold")
    ax.text(workers[0] + 0.3, tiempos[0], f"{tiempos[0]:.1f} s", color=INK, fontsize=10, va="center")
    ax.text(workers[-1], tiempos[-1] - max(tiempos) * 0.06, f"{tiempos[-1]:.1f} s",
            color=INK, fontsize=10, ha="center")
    ax.set_xticks(workers)
    ax.set_xlim(0.3, max(workers) + 0.7)
    ax.set_ylim(0, max(tiempos) * 1.12)
    ax.set_xlabel("Número de workers")
    ax.set_ylabel("Tiempo de entrenamiento (s)")
    estilo(ax)
    titulo(ax, "Tiempo de entrenamiento según el número de workers",
           f"Media recortada por cada número de workers · secuencial: {filas[0]['baseline']:.1f} s")
    fig.tight_layout()
    fig.savefig(os.path.join(out, "grafico4_tiempo_vs_workers.png"))
    plt.close(fig)


def grafico_ganancia(filas, threshold, out):
    datos = [r for r in filas if r["ganancia"] is not None]
    workers = [r["workers"] for r in datos]
    ganancias = [r["ganancia"] for r in datos]
    colores = [TEAL if g >= threshold else GRAY for g in ganancias]

    fig, ax = plt.subplots(figsize=(7.5, 4.3), dpi=200)
    ax.bar(workers, ganancias, color=colores, width=0.7)
    ax.axhline(threshold, color=INK, linewidth=1.2, linestyle=(0, (5, 4)))
    ax.text(max(workers) - 0.6, threshold + max(ganancias) * 0.015, f"umbral {threshold:g}%",
            color=INK, fontsize=10, ha="right", va="bottom")
    ax.axhline(0, color=GRID, linewidth=1)
    for w, g in zip(workers, ganancias):
        off = 1.0 if g >= 0 else -1.0
        ax.text(w, g + off, f"{g:.1f}", ha="center", va="bottom" if g >= 0 else "top",
                color=INK, fontsize=8)
    ax.set_xticks(workers)
    ax.set_xlim(min(workers) - 0.7, max(workers) + 0.7)
    lo = min(0, min(ganancias)) * 1.25
    ax.set_ylim(lo - 2 if lo < 0 else 0, max(ganancias) * 1.15)
    ax.set_xlabel("Workers (ganancia al pasar de n-1 a n workers)")
    ax.set_ylabel("Reducción del tiempo (%)")
    estilo(ax)
    titulo(ax, "Ganancia marginal de cada worker agregado",
           f"Turquesa: reduce el tiempo {threshold:g}% o más · gris: por debajo del umbral")
    fig.tight_layout()
    fig.savefig(os.path.join(out, "grafico5_ganancia_marginal.png"))
    plt.close(fig)


def main():
    parser = argparse.ArgumentParser(description="Gráficos del barrido fino de workers (HydroForest)")
    parser.add_argument("--csv", default=os.path.join("go", "sweep.csv"))
    parser.add_argument("--out", default=os.path.join("docs", "PC2", "figuras"))
    parser.add_argument("--threshold", type=float, default=5.0,
                        help="mismo umbral usado en -threshold del modo sweep")
    args = parser.parse_args()

    filas = leer_sweep(args.csv)
    os.makedirs(args.out, exist_ok=True)
    grafico_tiempo(filas, args.out)
    grafico_ganancia(filas, args.threshold, args.out)
    print(f"Gráficos guardados en {args.out}")


if __name__ == "__main__":
    main()

import argparse
import csv
import math
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


def leer_resultados(path):
    seq = con = None
    escal = []
    with open(path, newline="", encoding="utf-8") as f:
        for row in csv.DictReader(f):
            fila = {
                "workers": int(row["workers"]),
                "tiempo": float(row["media_recortada_s"]),
                "speedup": float(row["speedup"]),
                "eficiencia": float(row["eficiencia"]),
            }
            if row["seccion"] == "secuencial":
                seq = fila
            elif row["seccion"] == "concurrente":
                con = fila
            elif row["seccion"] == "escalabilidad":
                escal.append(fila)
    if seq is None or con is None or not escal:
        raise SystemExit(f"{path} no tiene las secciones esperadas (secuencial, concurrente, escalabilidad)")
    escal.sort(key=lambda r: r["workers"])
    return seq, con, escal


def estilo(ax):
    ax.grid(axis="y", color=GRID, linewidth=1)
    ax.set_axisbelow(True)
    ax.spines["left"].set_visible(False)
    ax.tick_params(length=0)


def titulo(ax, texto, subtitulo):
    ax.set_title(texto, loc="left", color=INK, fontsize=13, fontweight="bold", pad=22)
    ax.text(0, 1.04, subtitulo, transform=ax.transAxes, color=MUTED, fontsize=10)


def grafico_tiempos(seq, con, out):
    fig, ax = plt.subplots(figsize=(7, 4.2), dpi=200)
    labels = [f"Secuencial\n({seq['workers']} worker)", f"Concurrente\n({con['workers']} workers)"]
    vals = [seq["tiempo"], con["tiempo"]]
    bars = ax.bar(labels, vals, color=[GRAY, TEAL], width=0.5)
    top = max(vals)
    for b, v in zip(bars, vals):
        ax.text(b.get_x() + b.get_width() / 2, v + top * 0.02, f"{v:.2f} s",
                ha="center", va="bottom", color=INK, fontsize=12, fontweight="bold")
    ax.set_ylabel("Media recortada (segundos)")
    ax.set_ylim(0, top * 1.15)
    estilo(ax)
    speedup = seq["tiempo"] / con["tiempo"]
    titulo(ax, "Tiempo de entrenamiento: secuencial vs. concurrente", f"Speedup = {speedup:.2f}x")
    fig.tight_layout()
    fig.savefig(os.path.join(out, "grafico1_tiempos.png"))
    plt.close(fig)


def grafico_speedup(escal, out):
    workers = [r["workers"] for r in escal]
    speedup = [r["speedup"] for r in escal]
    wmax = max(workers)
    ymax = math.ceil(max(speedup)) + 3

    fig, ax = plt.subplots(figsize=(7, 4.2), dpi=200)
    ax.plot([1, ymax], [1, ymax], color=GRAY, linewidth=2, linestyle=(0, (5, 4)))
    ax.text(ymax * 0.66, ymax * 0.9, "Ideal (lineal)", ha="right", color=MUTED, fontsize=10)
    ax.plot(workers, speedup, color=TEAL, linewidth=2, marker="o", markersize=8,
            markeredgecolor="white", markeredgewidth=2, zorder=3)
    for w, s in zip(workers, speedup):
        if w == wmax:
            ax.text(w, s - ymax * 0.07, f"{s:.2f}x", ha="center", color=INK, fontsize=10, fontweight="bold")
        else:
            ax.text(w + wmax * 0.022, s - ymax * 0.045, f"{s:.2f}x", ha="left", color=INK, fontsize=10, fontweight="bold")
    ax.text(wmax, speedup[-1] + ymax * 0.045, "Medido", ha="right", color=TEAL, fontsize=10, fontweight="bold")
    ax.set_xticks(workers)
    ax.set_xlim(0, wmax + 1)
    ax.set_ylim(0, ymax)
    ax.set_xlabel("Número de workers")
    ax.set_ylabel("Speedup (T-Secuencial / T-Concurrente)")
    estilo(ax)
    titulo(ax, "Speedup según el número de workers",
           f"La línea ideal continúa hasta {wmax}x con {wmax} workers (fuera del gráfico)")
    fig.tight_layout()
    fig.savefig(os.path.join(out, "grafico2_speedup.png"))
    plt.close(fig)


def grafico_eficiencia(escal, out):
    workers = [r["workers"] for r in escal]
    eff = [r["eficiencia"] for r in escal]
    fig, ax = plt.subplots(figsize=(7, 4.2), dpi=200)
    xpos = list(range(len(workers)))
    bars = ax.bar(xpos, eff, color=TEAL, width=0.55)
    for b, e in zip(bars, eff):
        ax.text(b.get_x() + b.get_width() / 2, e - 0.03, f"{e:.2f}",
                ha="center", va="top", color="white", fontsize=10, fontweight="bold")
    ax.axhline(1.0, color=GRAY, linewidth=1.5, linestyle=(0, (5, 4)))
    ax.text(len(workers) - 0.65, 1.03, "Eficiencia ideal", ha="right", color=MUTED, fontsize=10)
    ax.set_xticks(xpos)
    ax.set_xticklabels([str(w) for w in workers])
    ax.set_ylim(0, max(1.18, max(eff) + 0.12))
    ax.set_xlabel("Número de workers")
    ax.set_ylabel("Eficiencia (speedup / workers)")
    estilo(ax)
    titulo(ax, "Eficiencia por worker", "Eficiencia = speedup / número de workers")
    fig.tight_layout()
    fig.savefig(os.path.join(out, "grafico3_eficiencia.png"))
    plt.close(fig)


def main():
    parser = argparse.ArgumentParser(description="Gráficos de speedup de HydroForest (PC2)")
    parser.add_argument("--csv", default=os.path.join("go", "results.csv"))
    parser.add_argument("--out", default=os.path.join("docs", "PC2", "figuras"))
    args = parser.parse_args()

    seq, con, escal = leer_resultados(args.csv)
    os.makedirs(args.out, exist_ok=True)
    grafico_tiempos(seq, con, args.out)
    grafico_speedup(escal, args.out)
    grafico_eficiencia(escal, args.out)
    print(f"Gráficos guardados en {args.out}")


if __name__ == "__main__":
    main()

"""
Prototipo de referencia del motor de anomalías (Python/pandas).
Sirve como especificación para portarlo a Go: mismas reglas, mismos umbrales.
Uso: python analysis_prototype.py readings.csv events.csv
"""
import sys, json
import pandas as pd

# --- Umbrales calibrados con el dataset ---
BASELINE_END = "2026-09-08"      # baseline = primera semana (01–07 sep)
DEV_THRESHOLD = 0.30             # desviación horaria vs baseline (normales llegan a ~22%)
V_MIN, V_MAX = 209.0, 231.0      # mas o menos 5% sobre 220 V nominal
RATIO_TOL = 0.50                 # tolerancia kWh vs V·I·FP (relativa al ratio propio del medidor)
EVENT_WINDOW_H = 3               # evento "coincide" si cae a ±3 h del inicio de la desviación
PF_DROP = 0.10                   # caída de factor de potencia que cuenta como cambio eléctrico
I_RISE = 0.30                    # subida de corriente que corrobora el aumento de consumo

readings = pd.read_csv(sys.argv[1] if len(sys.argv) > 1 else "readings.csv", parse_dates=["timestamp"])
events = pd.read_csv(sys.argv[2] if len(sys.argv) > 2 else "events.csv", parse_dates=["event_timestamp"])

df = readings.sort_values("timestamp").copy()
df["hour"] = df.timestamp.dt.hour
df["ratio"] = df.consumption_kwh / (df.voltage_v * df.current_a * df.power_factor / 1000)

# 1) Baseline por medidor y hora del día (mediana = robusta a outliers)
bl = df[df.timestamp < BASELINE_END]
base = bl.groupby(["meter_id", "hour"]).agg(
    b_kwh=("consumption_kwh", "median"), b_i=("current_a", "median"), b_pf=("power_factor", "median")
).reset_index()
df = df.merge(base, on=["meter_id", "hour"])
df["dev"] = df.consumption_kwh / df.b_kwh - 1
meter_ratio = bl.groupby("meter_id").ratio.median()

# 2) Calidad de datos: voltaje fuera de banda, FP imposible o energía incoherente con V·I·FP
df["dq"] = (
    (df.voltage_v < V_MIN) | (df.voltage_v > V_MAX) | (df.power_factor > 1) | (df.power_factor <= 0)
    | ((df.ratio / df.meter_id.map(meter_ratio) - 1).abs() > RATIO_TOL)
)

results = []
for meter, s in df.groupby("meter_id"):
    daily = s.groupby(s.timestamp.dt.date).consumption_kwh.sum()
    base_day = daily.iloc[:7].median()
    last_day = daily.iloc[-1]
    variation = (last_day / base_day - 1) * 100
    flagged = s[s.dev.abs() > DEV_THRESHOLD]
    dq = s[s.dq]
    ev = events[events.meter_id == meter]

    r = {"meter_id": meter, "baseline_kwh_day": round(base_day, 1), "current_kwh_day": round(last_day, 1),
         "variation_pct": round(variation, 1), "anomaly": False, "type": "NORMAL", "severity": "NONE",
         "confidence": None, "evidence": {}}

    # 3) Clasificación (orden importa: calidad de datos primero)
    if len(dq) >= 3:
        r.update(anomaly=True, type="DATA_QUALITY", severity="HIGH")
        dq_events = ev[ev.event_type == "DATA_QUALITY"]
        r["evidence"] = {
            "invalid_readings": int(len(dq)), "first_invalid": str(dq.timestamp.min()),
            "voltage_range": [float(dq.voltage_v.min()), float(dq.voltage_v.max())],
            "power_factor_values": sorted(dq.power_factor.unique().tolist()),
            "consumption_stable": bool(abs(variation) < 5),
            "corroborating_event": dq_events.description.tolist(),
        }
        r["confidence"] = round(min(0.99, 0.75 + 0.1 * r["evidence"]["consumption_stable"]
                                     + 0.1 * (len(dq_events) > 0)), 2)
    elif len(flagged) > 0:
        start, end = flagged.timestamp.min(), flagged.timestamp.max()
        duration_h = len(flagged)
        direction = "UP" if flagged.dev.mean() > 0 else "DOWN"
        near = ev[(ev.event_timestamp - start).abs() <= pd.Timedelta(hours=EVENT_WINDOW_H)]
        explaining = near[near.event_type.isin(["OPERATIONAL_CHANGE", "SCHEDULED_OUTAGE"])]
        i_change = (flagged.current_a / flagged.b_i - 1).mean()
        pf_change = (flagged.power_factor - flagged.b_pf).mean()
        r["evidence"] = {
            "start": str(start), "end": str(end), "hours_affected": duration_h, "direction": direction,
            "mean_hourly_deviation_pct": round(flagged.dev.mean() * 100, 1),
            "current_change_pct": round(i_change * 100, 1), "power_factor_change": round(pf_change, 3),
            "related_events": near[["event_type", "description"]].to_dict("records"),
        }
        r["anomaly"] = True
        if len(explaining):
            etype = explaining.iloc[0].event_type
            if etype == "SCHEDULED_OUTAGE":
                r.update(type="FALSE_POSITIVE", severity="LOW", anomaly=False)
                r["confidence"] = 0.85 + 0.1 * (duration_h <= 24)
            else:
                r.update(type="EXPLAINABLE_ANOMALY", severity="MEDIUM")
                r["confidence"] = 0.8 + 0.1 * (i_change > I_RISE * 0.5)
        else:
            r.update(type="REAL_ANOMALY", severity="HIGH")
            r["confidence"] = (0.7 + 0.1 * (i_change > I_RISE) + 0.1 * (pf_change < -PF_DROP)
                               + 0.05 * (duration_h >= 24) + 0.03 * (abs(variation) > 50))
        r["confidence"] = round(min(r["confidence"], 0.99), 2)
    results.append(r)

# 4) Prioridad: severidad × confianza × magnitud
W = {"HIGH": 3, "MEDIUM": 2, "LOW": 1, "NONE": 0}
for r in results:
    mag = 1 + abs(r["variation_pct"]) / 100
    r["priority_score"] = round(W[r["severity"]] * (r["confidence"] or 0) * mag, 2)
results.sort(key=lambda r: -r["priority_score"])

for r in results:
    print(f'{r["meter_id"]}  {r["type"]:<20} {r["severity"]:<7} conf={r["confidence"]}  '
          f'var={r["variation_pct"]:+.1f}%  prio={r["priority_score"]}')
with open("analysis_output.json", "w") as f:
    json.dump(results, f, indent=2, default=str, ensure_ascii=False)

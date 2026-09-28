# Arquitectura — AI Energy Management Platform

---

## 1. Vista general

![Diagrama](docs/diagrama-arquitectura.png)

| Contenedor | Qué hace | Por qué así |
|---|---|---|
| **frontend** | nginx sirve la SPA compilada y reenvía `/api` al backend. | Mismo origen para el navegador: no hace falta CORS en producción. |
| **backend** | API REST en Go. Orquesta el análisis, ejecuta el motor y llama a Claude. | Un solo binario, sin puerto expuesto al host; solo se accede por nginx. |
| **postgres** | Guarda medidores, lecturas, eventos, análisis y anomalías. | Consultas agregadas (medianas, sumas) directamente en SQL. |
| **seed** | Carga los CSV al arrancar, solo si la base está vacía (`-if-empty`). | La carga de datos no es un endpoint: es una tarea de operación, no de usuario. |
| **Claude** | Servicio externo. Redacta `reason` y `recommended_action`. | Si no responde, la plataforma usa una plantilla y sigue funcionando. |

Orden de arranque: `postgres` (healthcheck) → `seed` → `backend` (healthcheck `/health`) → `frontend`.

En **desarrollo local**, Vite sirve el frontend en `:5173` y llama al backend en `:8080` con CORS (`FRONTEND_ORIGIN`). Postgres se publica en el puerto `5433` del host.

---

## 2. Backend: paquetes y dependencias

```mermaid
flowchart TB
    subgraph cmd["cmd/ — ejecutables"]
        server["server<br/>API HTTP"]
        seedcmd["seed<br/>carga de CSV"]
    end

    subgraph internal["internal/"]
        httpapi["httpapi<br/>rutas, handlers, login demo,<br/>orquestación del análisis"]
        analysis["analysis<br/>MOTOR: baseline, detección,<br/>clasificación, confianza, prioridad"]
        ai["ai<br/>Claude + validación de cifras<br/>+ plantilla de respaldo"]
        db["db<br/>modelos GORM, constantes,<br/>conexión, AutoMigrate"]
        csvdata["csvdata<br/>lectura de data/*.csv"]
        config["config<br/>ÚNICA lectura del entorno .env"]
    end

    server --> httpapi
    server --> ai
    server --> config
    server --> db
    seedcmd --> csvdata
    seedcmd --> db
    seedcmd --> config
    httpapi --> analysis
    httpapi --> ai
    httpapi --> db
    httpapi --> config
    ai --> analysis
    ai --> config
    ai --> db
    analysis --> db
    csvdata --> db
    db --> config
```

| Paquete | Responsabilidad | 
|---|---|
| `analysis` | Función pura `Analyze(lecturas, eventos) → resultados`. Todas las decisiones: tipo, severidad, confianza, prioridad y evidencia. | 
| `ai` | Recibe un resultado del motor y devuelve el texto. Interfaz `Explainer` con dos implementaciones: `LLMExplainer` (Claude) y `TemplateExplainer`. |
| `httpapi` | Rutas Gin bajo `/api/v1`, autenticación demo (`Bearer`), consultas SQL para las pantallas y el análisis paso a paso. | 
| `db` | Modelos GORM, constantes (estados, severidades, pasos) y conexión. | 
| `config` | Carga `backend/.env` y valida las variables obligatorias. |
| `csvdata` | Lee los CSV. Lo usan el seed y los tests del motor. | 

**Por qué el motor es una función pura:** se puede probar con los CSV reales sin levantar nada. El test de aceptación (`analysis/acceptance_test.go`) comprueba los 4 casos de la prueba cada vez que se ejecuta `go test ./...`.

**Una sola fuente para las reglas:** los umbrales (30 %, 209–231 V, ventana de 3 h, eventos que explican) viven en `analysis`. El frontend los pide a `GET /api/v1/analysis/rules` en vez de copiarlos.

---

## 3. Flujo de Run AI Analysis

```mermaid
sequenceDiagram
    autonumber
    actor U as Usuario
    participant F as Frontend
    participant A as Backend (httpapi)
    participant M as Motor (analysis)
    participant E as Explicador (ai)
    participant C as Claude
    participant DB as PostgreSQL

    U->>F: Run AI Analysis → confirmar
    F->>A: POST /api/v1/ai/analyze
    A->>DB: crear analysis_run (RUNNING)
    A-->>F: run + lista de pasos
    Note over A: goroutine en segundo plano

    loop cada ~500 ms hasta COMPLETED o FAILED
        F->>A: GET /api/v1/ai/analysis/{id}
        A-->>F: current_step
    end

    A->>DB: 1. READINGS: leer lecturas y eventos
    A->>M: 2–5. Analyze(lecturas, eventos)
    Note over M: baseline por hora → horas desviadas<br/>y lecturas imposibles → correlación<br/>eléctrica → eventos → tipo, severidad,<br/>confianza, prioridad
    M-->>A: 12 resultados (8 NORMAL + 4 hallazgos)

    A->>E: 6. EXPLANATION: explicar los 4 hallazgos en paralelo
    par un hallazgo por goroutine
        E->>C: evidencia en JSON + prompt, salida con esquema fijo
        C-->>E: reason + recommended_action
        E->>E: ¿cada cifra está en la evidencia?
        alt válido
            Note over E: explanation_source = llm
        else error, timeout o cifra inventada
            Note over E: plantilla, explanation_source = template
        end
    end
    E-->>A: explicaciones

    A->>DB: 7. RECOMMENDATION: guardar 4 anomalías (OPEN)
    A->>DB: analysis_run → COMPLETED
    F->>F: refrescar todas las pantallas
```

- Cada paso espera `ANALYSIS_STEP_DELAY_MS` (700 ms por defecto) para que la barra de progreso se vea; sin esa pausa, los pasos 1–5 serían instantáneos.
- Solo se guardan los resultados distintos de `NORMAL`. Cada análisis crea filas nuevas, y las pantallas muestran siempre el último `COMPLETED`.
- Si el servidor se reinicia a mitad, al arrancar `db.FailInterruptedRuns` marca ese análisis como `FAILED` para que no quede colgado.

---

## 4. Motor: cómo decide

```mermaid
flowchart TB
    start(["Lecturas de un medidor"]) --> base["Baseline: mediana por hora del día<br/>de la primera semana, 01–07 sep"]
    base --> q1{"¿3 o más lecturas imposibles?<br/>voltaje fuera de 209–231 V,<br/>FP imposible o energía que no cuadra"}
    q1 -- sí --> dq["DATA_QUALITY · alta"]
    q1 -- no --> q2{"¿Alguna hora se aleja<br/>más de un 30 % de su baseline?"}
    q2 -- no --> normal["NORMAL"]
    q2 -- sí --> q3{"¿Evento a 3 h o menos<br/>del inicio de la desviación?"}
    q3 -- SCHEDULED_OUTAGE --> fp["FALSE_POSITIVE · baja"]
    q3 -- OPERATIONAL_CHANGE --> ea["EXPLAINABLE_ANOMALY · media"]
    q3 -- "ninguno, o UNKNOWN" --> real["REAL_ANOMALY · alta"]

    dq & fp & ea & real --> score["Confianza: base + pruebas, tope 0,99<br/>Prioridad: peso × confianza × (1 + variación)"]
```

---

## 5. Frontend

```mermaid
flowchart LR
    subgraph pages["pages/"]
        login["Login"]
        dash["Dashboard"]
        meters["Medidores"]
        detail["Detalle del medidor"]
        anomalies["Anomalías IA"]
        inv["Investigación"]
    end

    run["components/run-analysis<br/>confirmación + barra de pasos"]
    layout["components/app-layout<br/>menú lateral + cabecera"]
    query["TanStack Query<br/>caché, polling, invalidación"]
    apiClient["lib/api.ts<br/>fetch + token Bearer"]
    rules["lib/use-rules<br/>umbrales del motor"]
    backend["Backend /api/v1"]

    layout --> run
    pages --> query
    run --> query
    rules --> query
    query --> apiClient
    apiClient --> backend
```

- **Vite + React SPA:** la API es propia y no hay necesidad de SEO ni de renderizado en servidor.
- **TanStack Query** guarda en caché cada consulta. Al terminar un análisis, `run-analysis` invalida las consultas y todas las pantallas se actualizan solas.
- **Rutas con `lazy`:** cada página se descarga cuando se visita.
- **Sesión:** el token del login se guarda en el navegador; un `401` devuelve al login.

---

## 6. Modelo de datos

```mermaid
erDiagram
    METERS ||--o{ READINGS : "tiene"
    METERS ||--o{ EVENTS : "tiene"
    METERS ||--o{ ANOMALIES : "tiene"
    ANALYSIS_RUNS ||--o{ ANOMALIES : "produce"

    METERS {
        string meter_id PK
        string name
        string location
    }
    READINGS {
        uint id PK
        string meter_id "único con timestamp"
        timestamp timestamp
        float consumption_kwh
        float voltage_v
        float current_a
        float power_factor
    }
    EVENTS {
        uint id PK
        string meter_id
        timestamp event_timestamp
        string event_type "OPERATIONAL_CHANGE, SCHEDULED_OUTAGE, UNKNOWN, DATA_QUALITY"
        string description
    }
    ANALYSIS_RUNS {
        uint id PK
        string status "RUNNING, COMPLETED, FAILED"
        string current_step
        timestamp started_at
        timestamp finished_at
    }
    ANOMALIES {
        uint id PK
        uint analysis_run_id FK
        string meter_id
        bool is_anomaly
        string type
        string severity
        float confidence
        float priority_score
        float variation_pct
        string reason
        string recommended_action
        string explanation_source "llm o template"
        jsonb evidence
        string status "OPEN, INVESTIGATING, RESOLVED, DISMISSED"
    }
```

- El esquema se crea con `AutoMigrate` al conectar; no hay migraciones versionadas en el alcance del MVP.
- Las fechas se guardan como `timestamp` sin zona horaria, porque los CSV vienen en hora de planta, y el frontend las muestra tal cual.
- `evidence` (JSONB) guarda todo lo que el motor calculó para esa anomalía. Es lo que recibe Claude y lo que muestra "Evidencia completa".

---

## 7. API

Todas las rutas llevan el prefijo `/api/v1` y requieren `Authorization: Bearer <token>`, salvo el login. `GET /health` queda fuera del prefijo.

| Método | Ruta | Pantalla que la usa |
|---|---|---|
| POST | `/auth/login` | Login |
| GET | `/dashboard/summary` | Dashboard |
| GET | `/meters` | Dashboard, Medidores, Anomalías IA |
| GET | `/meters/{meterId}` | Detalle |
| GET | `/meters/{meterId}/readings` | Detalle, Investigación |
| GET | `/meters/{meterId}/baseline` | Detalle, Investigación |
| GET | `/events` | Detalle, Investigación |
| GET | `/analysis/rules` | Medidores, Detalle, Investigación |
| POST | `/ai/analyze` | Run AI Analysis |
| GET | `/ai/analysis/{id}` | Run AI Analysis (polling) |
| GET | `/anomalies` | Dashboard, Anomalías IA |
| GET | `/anomalies/{id}` | Detalle, Investigación |
| PATCH | `/anomalies/{id}` | Investigación (cambiar estado) |

---

## 8. Decisiones y siguientes pasos

| Decisión | Motivo |
|---|---|
| El motor decide y el LLM solo explica | Resultados reproducibles y auditables; la IA no puede inventar un problema. |
| Validación de cifras + plantilla de respaldo | La demo nunca depende de que la API de Claude responda. |
| Monolito modular | los paquetes ya separan responsabilidades |
| Análisis en goroutine + polling | Suficiente para un análisis de segundos; sin colas ni WebSockets. |


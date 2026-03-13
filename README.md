# API Contract (OpenAPI)

## CI/CD e Docker

- Pipeline GitHub Actions: `.github/workflows/ci-cd.yml`
- Documentacao de CI/CD, Docker Hub e variaveis: `docs/CI_CD_DOCKER.md`

Base path (server): `/api/v1`

## Authentication

- Security scheme: `ClientCredentials` (`apiKey` in header)
- Header defined: `X-Client-ID`
- Description also requires: `X-Client-Secret`
- Protected endpoints: all under `/projects`, `/groups`, `/users`, `/metrics/*`, `/issues*`
- Public endpoints (no auth): `/health`, `/metrics`

## Common Error Response

All referenced error responses use `ErrorResponse`:

- `code: string`
- `message: string`
- `request_id: string`

## Endpoints

### GET /projects

- Headers: `X-Client-ID`, `X-Client-Secret`
- Query params:
  - `search: string` (optional)
  - `group_path: string` (optional)
- `200`: `Project[]`
- `401`: `ErrorResponse`
- `500`: `ErrorResponse`

### GET /groups

- Headers: `X-Client-ID`, `X-Client-Secret`
- Query params:
  - `search: string` (optional)
- `200`: `Group[]`
- `401`: `ErrorResponse`
- `500`: `ErrorResponse`

### GET /users

- Headers: `X-Client-ID`, `X-Client-Secret`
- Query params:
  - `search: string` (optional)
  - `project_id: integer` (optional)
  - `group_path: string` (optional)
- `200`: `User[]`
- `401`: `ErrorResponse`
- `422`: `ErrorResponse`
- `500`: `ErrorResponse`

### GET /metrics/delivery

- Headers: `X-Client-ID`, `X-Client-Secret`
- Query params:
  - `start_date: string(date)` (required)
  - `end_date: string(date)` (required)
  - `project_id: integer` (optional)
  - `group_path: string` (optional)
  - `assignee: string` (optional)
- `200`: `DeliveryMetricsResponse`
- `400`: `ErrorResponse`
- `401`: `ErrorResponse`
- `422`: `ErrorResponse`
- `500`: `ErrorResponse`

### GET /metrics/quality

- Headers: `X-Client-ID`, `X-Client-Secret`
- Query params:
  - `start_date: string(date)` (required)
  - `end_date: string(date)` (required)
  - `project_id: integer` (optional)
  - `group_path: string` (optional)
- `200`: `QualityMetricsResponse`
- `400`: `ErrorResponse`
- `401`: `ErrorResponse`
- `422`: `ErrorResponse`
- `500`: `ErrorResponse`

### GET /metrics/wip

- Headers: `X-Client-ID`, `X-Client-Secret`
- Query params:
  - `project_id: integer` (optional)
  - `group_path: string` (optional)
  - `assignee: string` (optional)
- `200`: `WipMetricsResponse`
- `401`: `ErrorResponse`
- `422`: `ErrorResponse`
- `500`: `ErrorResponse`

### GET /issues

- Headers: `X-Client-ID`, `X-Client-Secret`
- Query params:
  - `start_date: string(date)` (optional)
  - `end_date: string(date)` (optional)
  - `project_id: integer` (optional)
  - `group_path: string` (optional)
  - `assignee: string` (optional)
  - `state: string` (optional)
  - `metric_flag: string` (optional, enum: `rework | blocked | aging_wip | bypass`)
  - `page: integer` (optional, min `1`, default `1`)
  - `page_size: integer` (optional, min `1`, max `100`, default `25`)
- `200`: `IssuesListResponse`
- `400`: `ErrorResponse`
- `401`: `ErrorResponse`
- `422`: `ErrorResponse`
- `500`: `ErrorResponse`

### GET /issues/{id}/timeline

- Headers: `X-Client-ID`, `X-Client-Secret`
- Path params:
  - `id: integer` (required)
- `200`: `IssueTimelineResponse`
- `400`: `ErrorResponse`
- `401`: `ErrorResponse`
- `404`: `ErrorResponse`
- `500`: `ErrorResponse`

### GET /health

- Authentication: none
- Query params: none
- `200`: `{ status: string }`
- `503`: `{ status: string, error: string }`

### GET /metrics

- Authentication: none
- Query params: none
- `200`: `MetricsSnapshot`

### GET /users/{username}/performance

- Headers: `X-Client-ID`, `X-Client-Secret`
- Path params:
  - `username: string` (required)
- Query params:
  - `start_date: string(date)` (optional)
  - `end_date: string(date)` (optional)
  - `project_id: integer` (optional)
  - `group_path: string` (optional)
- `200`: `UserPerformanceResponse`
- `400`: `ErrorResponse`
- `401`: `ErrorResponse`
- `404`: `ErrorResponse`
- `422`: `ErrorResponse`
- `500`: `ErrorResponse`

## JSON Examples (capturados via curl)

Headers usados nos endpoints protegidos:

```bash
curl -H "X-Client-ID: myclient" \
     -H "X-Client-Secret: mysecret" \
     http://localhost:8080/api/v1/projects
```

Os exemplos abaixo sao **corpos completos reais**, capturados com `curl` na API local.

### GET /health

Request:

```bash
curl http://localhost:8080/health
```

```json
{
  "status": "healthy"
}
```

### GET /metrics

Request:

```bash
curl http://localhost:8080/metrics
```

```json
{
  "endpoints": [
    {
      "method": "GET",
      "path": "/api/v1/issues/1/timeline",
      "total_requests": 2,
      "status_counts": {
        "200": 2
      },
      "error_count": 0,
      "avg_latency": "6.965145ms",
      "p95_latency": "8.429906ms"
    },
    {
      "method": "GET",
      "path": "/projects",
      "total_requests": 1,
      "status_counts": {
        "404": 1
      },
      "error_count": 1,
      "avg_latency": "87.221µs",
      "p95_latency": "87.221µs"
    },
    {
      "method": "GET",
      "path": "/api/v1/health",
      "total_requests": 1,
      "status_counts": {
        "404": 1
      },
      "error_count": 1,
      "avg_latency": "27.697µs",
      "p95_latency": "27.697µs"
    },
    {
      "method": "GET",
      "path": "/api/v1/metrics",
      "total_requests": 1,
      "status_counts": {
        "404": 1
      },
      "error_count": 1,
      "avg_latency": "32.597µs",
      "p95_latency": "32.597µs"
    },
    {
      "method": "GET",
      "path": "/api/v1/projects",
      "total_requests": 5,
      "status_counts": {
        "200": 4,
        "401": 1
      },
      "error_count": 1,
      "avg_latency": "46.322944ms",
      "p95_latency": "71.13265ms"
    },
    {
      "method": "GET",
      "path": "/api/v1/groups",
      "total_requests": 5,
      "status_counts": {
        "200": 5
      },
      "error_count": 0,
      "avg_latency": "50.701361ms",
      "p95_latency": "90.787607ms"
    },
    {
      "method": "GET",
      "path": "/api/v1/metrics/delivery",
      "total_requests": 3,
      "status_counts": {
        "200": 3
      },
      "error_count": 0,
      "avg_latency": "205.929128ms",
      "p95_latency": "272.001267ms"
    },
    {
      "method": "GET",
      "path": "/api/v1/metrics/quality",
      "total_requests": 3,
      "status_counts": {
        "200": 3
      },
      "error_count": 0,
      "avg_latency": "237.527362ms",
      "p95_latency": "302.990015ms"
    },
    {
      "method": "GET",
      "path": "/api/v1/metrics/wip",
      "total_requests": 3,
      "status_counts": {
        "200": 3
      },
      "error_count": 0,
      "avg_latency": "104.002398ms",
      "p95_latency": "151.036307ms"
    },
    {
      "method": "GET",
      "path": "/api/v1/issues",
      "total_requests": 6,
      "status_counts": {
        "200": 6
      },
      "error_count": 0,
      "avg_latency": "123.891814ms",
      "p95_latency": "178.768306ms"
    },
    {
      "method": "GET",
      "path": "/api/v1/issues/2/timeline",
      "total_requests": 1,
      "status_counts": {
        "200": 1
      },
      "error_count": 0,
      "avg_latency": "3.954879ms",
      "p95_latency": "3.954879ms"
    },
    {
      "method": "GET",
      "path": "/health",
      "total_requests": 3,
      "status_counts": {
        "200": 3
      },
      "error_count": 0,
      "avg_latency": "2.31589ms",
      "p95_latency": "5.751456ms"
    },
    {
      "method": "GET",
      "path": "/metrics",
      "total_requests": 2,
      "status_counts": {
        "200": 2
      },
      "error_count": 0,
      "avg_latency": "169.074µs",
      "p95_latency": "177.375µs"
    },
    {
      "method": "GET",
      "path": "/api/v1/users",
      "total_requests": 3,
      "status_counts": {
        "200": 3
      },
      "error_count": 0,
      "avg_latency": "647.739802ms",
      "p95_latency": "861.978789ms"
    }
  ],
  "total_requests": 39,
  "total_errors": 4
}
```

### GET /projects

Request:

```bash
curl -H "X-Client-ID: myclient" \
     -H "X-Client-Secret: mysecret" \
     "http://localhost:8080/api/v1/projects?search=Atlantaza"
```

```json
[
  {
    "id": 19,
    "name": "Atlantaza",
    "path": "apps-android/atlantaza",
    "total_issues": 0,
    "last_synced_at": "2026-03-08T00:00:01.838855Z"
  },
  {
    "id": 20,
    "name": "ServiceAtlantazaapp",
    "path": "webapis/serviceatlantazaapp",
    "total_issues": 0,
    "last_synced_at": "2026-03-07T23:58:09.04036Z"
  }
]
```

### GET /groups

Request:

```bash
curl -H "X-Client-ID: myclient" \
     -H "X-Client-Secret: mysecret" \
     "http://localhost:8080/api/v1/groups?search=apps-android"
```

```json
[
  {
    "group_path": "apps-android",
    "project_count": 16,
    "total_issues": 0,
    "last_synced_at": "2026-03-08T00:00:39.804552Z"
  }
]
```

### GET /users

Request:

```bash
curl -H "X-Client-ID: myclient" \
     -H "X-Client-Secret: mysecret" \
     "http://localhost:8080/api/v1/users?search=arthurdue"
```

```json
[
  {
    "username": "arthurdue",
    "display_name": "arthurdue",
    "active_issues": 6,
    "completed_issues_last_30_days": 0
  }
]
```

### GET /metrics/delivery

Request:

```bash
curl -H "X-Client-ID: myclient" \
     -H "X-Client-Secret: mysecret" \
     "http://localhost:8080/api/v1/metrics/delivery"
```

```json
{
  "period": {},
  "throughput": {
    "total_issues_done": 1123,
    "avg_per_week": 40.94
  },
  "speed_metrics_days": {
    "lead_time": {
      "avg": 20.871322721875927,
      "p85": 37.614333333333285
    },
    "cycle_time": {
      "avg": 15.745159542891066,
      "p85": 30.919833333333333
    }
  },
  "breakdown_by_assignee": [
    {
      "assignee": "nevez",
      "issues_delivered": 107,
      "avg_cycle_time": 15.473578660436138
    },
    {
      "assignee": "ianfelps",
      "issues_delivered": 103,
      "avg_cycle_time": 10.280214401294499
    },
    {
      "assignee": "torezan",
      "issues_delivered": 97,
      "avg_cycle_time": 15.951632302405498
    },
    {
      "assignee": "gabriel",
      "issues_delivered": 95,
      "avg_cycle_time": 14.737833333333334
    },
    {
      "assignee": "danilo",
      "issues_delivered": 88,
      "avg_cycle_time": 11.108593749999999
    },
    {
      "assignee": "vitorfsampaio",
      "issues_delivered": 61,
      "avg_cycle_time": 16.58762295081967
    },
    {
      "assignee": "sarah",
      "issues_delivered": 59,
      "avg_cycle_time": 18.490755649717514
    },
    {
      "assignee": "bezerra",
      "issues_delivered": 58,
      "avg_cycle_time": 15.83421695402299
    },
    {
      "assignee": "maria_dev",
      "issues_delivered": 49,
      "avg_cycle_time": 24.976836734693876
    },
    {
      "assignee": "teresio",
      "issues_delivered": 37,
      "avg_cycle_time": 7.701306306306306
    },
    {
      "assignee": "fabio",
      "issues_delivered": 36,
      "avg_cycle_time": 7.454606481481481
    },
    {
      "assignee": "ludwin",
      "issues_delivered": 35,
      "avg_cycle_time": 26.836083333333335
    },
    {
      "assignee": "calebariel",
      "issues_delivered": 32,
      "avg_cycle_time": 9.471302083333333
    },
    {
      "assignee": "cleslley",
      "issues_delivered": 29,
      "avg_cycle_time": 21.323778735632185
    },
    {
      "assignee": "elismardev",
      "issues_delivered": 26,
      "avg_cycle_time": 11.361778846153847
    },
    {
      "assignee": "viniciuseloicorrea9",
      "issues_delivered": 26,
      "avg_cycle_time": 18.21213141025641
    },
    {
      "assignee": "vinijr",
      "issues_delivered": 23,
      "avg_cycle_time": 14.697699275362318
    },
    {
      "assignee": "vinentee",
      "issues_delivered": 22,
      "avg_cycle_time": 20.184829545454544
    },
    {
      "assignee": "quiasz",
      "issues_delivered": 21,
      "avg_cycle_time": 17.51888888888889
    },
    {
      "assignee": "nelsonlima1989",
      "issues_delivered": 19,
      "avg_cycle_time": 20.341885964912283
    },
    {
      "assignee": "lina",
      "issues_delivered": 17,
      "avg_cycle_time": 16.03235294117647
    },
    {
      "assignee": "gloria",
      "issues_delivered": 15,
      "avg_cycle_time": 23.718833333333333
    },
    {
      "assignee": "arthurdue",
      "issues_delivered": 13,
      "avg_cycle_time": 19.015865384615385
    },
    {
      "assignee": "rubens",
      "issues_delivered": 8,
      "avg_cycle_time": 32.91786458333333
    },
    {
      "assignee": "eduardo_frois",
      "issues_delivered": 8,
      "avg_cycle_time": 11.484843750000001
    },
    {
      "assignee": "leony99",
      "issues_delivered": 7,
      "avg_cycle_time": 45.487678571428575
    },
    {
      "assignee": "edsonpinheiro",
      "issues_delivered": 6,
      "avg_cycle_time": 22.147499999999997
    },
    {
      "assignee": "igor",
      "issues_delivered": 5,
      "avg_cycle_time": 4.09425
    },
    {
      "assignee": "idevilson",
      "issues_delivered": 4,
      "avg_cycle_time": 2.3334375
    },
    {
      "assignee": "felipe",
      "issues_delivered": 1
    },
    {
      "assignee": "gabriel_diniz",
      "issues_delivered": 1,
      "avg_cycle_time": 0.25625000000000003
    }
  ]
}
```

### GET /metrics/quality

Request:

```bash
curl -H "X-Client-ID: myclient" \
     -H "X-Client-Secret: mysecret" \
     "http://localhost:8080/api/v1/metrics/quality"
```

```json
{
  "rework": {
    "ping_pong_rate_pct": 11.534391534391535,
    "total_reworked_issues": 218,
    "avg_rework_cycles_per_issue": 0.12645502645502646
  },
  "process_health": {
    "bypass_rate_pct": 24.04274265360641,
    "first_time_pass_rate_pct": 75.95725734639359
  },
  "bottlenecks": {
    "total_blocked_time_hours": 60893.7,
    "avg_blocked_time_per_issue_hours": 1171.0326923076923
  },
  "defects": {
    "bug_ratio_pct": 0.052910052910052914
  }
}
```

### GET /metrics/wip

Request:

```bash
curl -H "X-Client-ID: myclient" \
     -H "X-Client-Secret: mysecret" \
     "http://localhost:8080/api/v1/metrics/wip?project_id=10"
```

```json
{
  "current_wip": {
    "in_progress": 1,
    "qa_review": 4
  },
  "aging_wip": [
    {
      "issue_iid": 32,
      "title": "[DERR - WEB] [Grupo Infração] Alguns usuarios e imei nao aparecem. Aparecem em branco. Isso no sistema antigo",
      "current_state": "IN_PROGRESS",
      "days_in_state": 174,
      "warning_flag": true
    },
    {
      "issue_iid": 6,
      "title": "[DERR - WEB] [Consulta Infraçoes] Ta faltando o campo de Endereço e KM no novo",
      "assignees": [
        "quiasz"
      ],
      "current_state": "QA_REVIEW",
      "days_in_state": 152,
      "warning_flag": true
    },
    {
      "issue_iid": 39,
      "title": "[DERR WEB] - Correção na impressão do BRAT com opção “Agrupar Consulta",
      "assignees": [
        "vinijr"
      ],
      "current_state": "QA_REVIEW",
      "days_in_state": 146,
      "warning_flag": true
    },
    {
      "issue_iid": 38,
      "title": "[DERRJ - WEB] - Remover módulo “Consulta Credenciais",
      "assignees": [
        "vinijr"
      ],
      "current_state": "QA_REVIEW"
    },
    {
      "issue_iid": 60,
      "title": "CORREÇÃO: Módulo de inconsistência",
      "assignees": [
        "nevez"
      ],
      "current_state": "QA_REVIEW"
    }
  ]
}
```

### GET /issues

Request:

```bash
curl -H "X-Client-ID: myclient" \
     -H "X-Client-Secret: mysecret" \
     "http://localhost:8080/api/v1/issues?page=1&page_size=1"
```

```json
{
  "items": [
    {
      "id": 1,
      "project_id": 93,
      "issue_iid": 1,
      "title": "Componente Modal Dinâmico",
      "assignees": [
        "danilo"
      ],
      "current_canonical_state": "UNKNOWN"
    }
  ],
  "page": 1,
  "page_size": 1,
  "total": 1890
}
```

### GET /issues/{id}/timeline

Request:

```bash
curl -H "X-Client-ID: myclient" \
     -H "X-Client-Secret: mysecret" \
     "http://localhost:8080/api/v1/issues/1/timeline"
```

```json
{
  "issue": {
    "gitlab_iid": 1,
    "project_id": 93,
    "id": 1,
    "title": "Componente Modal Dinâmico",
    "assignees": [
      "danilo"
    ],
    "current_canonical_state": "UNKNOWN",
    "gitlab_created_at": "2023-12-12T14:49:58.651Z"
  },
  "timeline": [
    {
      "type": "issue_created",
      "timestamp": "2023-12-12T14:49:58.651Z",
      "to_state": "OPEN"
    },
    {
      "type": "comment",
      "timestamp": "2023-12-12T15:22:32.508Z",
      "actor": "Leony99",
      "body": "@danilo @igorcoutinho testando fazer Tasks dentro do Gitlab"
    }
  ]
}
```

## Schemas (Response Contracts)

- `Project`: `id`, `name`, `path`, `total_issues`, `last_synced_at`
- `Group`: `group_path`, `project_count`, `total_issues`, `last_synced_at`
- `User`: `username`, `display_name`, `active_issues`, `completed_issues_last_30_days`
- `DeliveryMetricsResponse`: `period`, `throughput`, `speed_metrics_days`, `breakdown_by_assignee`
- `QualityMetricsResponse`: `rework`, `process_health`, `bottlenecks`, `defects`
- `WipMetricsResponse`: `current_wip`, `aging_wip`
- `IssuesListResponse`: `items`, `page`, `page_size`, `total`
- `IssueTimelineResponse`: `issue`, `timeline`
- `MetricsSnapshot`: `endpoints`, `total_requests`, `total_errors`
- `UserPerformanceResponse`: `user`, `period`, `delivery`, `quality`, `wip`

### GET /users/{username}/performance

Request:

```bash
curl -H "X-Client-ID: myclient" \
     -H "X-Client-Secret: mysecret" \
     "http://localhost:8080/api/v1/users/john.doe/performance?start_date=2026-01-01&end_date=2026-03-01"
```

```json
{
  "user": {
    "username": "john.doe",
    "display_name": "John Doe",
    "active_issues": 5,
    "completed_issues_last_30_days": 12
  },
  "period": {
    "start_date": "2026-01-01",
    "end_date": "2026-03-01"
  },
  "delivery": {
    "throughput": {
      "total_issues_done": 45,
      "avg_per_week": 3.75
    },
    "speed_metrics_days": {
      "lead_time": {
        "avg": 18.5,
        "p85": 35.2
      },
      "cycle_time": {
        "avg": 12.3,
        "p85": 25.8
      }
    }
  },
  "quality": {
    "rework": {
      "ping_pong_rate_pct": 8.5,
      "total_reworked_issues": 4,
      "avg_rework_cycles_per_issue": 0.12
    },
    "ghost_work": {
      "rate_pct": 5.2
    },
    "process_health": {
      "bypass_rate_pct": 3.1,
      "first_time_pass_rate_pct": 91.5
    },
    "bottlenecks": {
      "total_blocked_time_hours": 156.5,
      "avg_blocked_time_per_issue_hours": 8.2
    },
    "defects": {
      "bug_ratio_pct": 2.1
    }
  },
  "wip": {
    "current_wip": {
      "in_progress": 3,
      "qa_review": 2,
      "blocked": 0
    },
    "aging_wip": [
      {
        "issue_iid": 42,
        "title": "Fix authentication bug",
        "assignees": ["john.doe"],
        "current_state": "IN_PROGRESS",
        "days_in_state": 15,
        "warning_flag": true
      }
    ]
  }
}
```

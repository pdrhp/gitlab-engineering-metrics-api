# Database Examples

## Objetivo

Este documento mostra exemplos representativos de dados para facilitar o entendimento de colunas `jsonb`, relacoes entre tabelas e como os dados do banco sustentam as respostas da API.

Os exemplos abaixo sao ilustrativos, mas seguem a estrutura real definida em `db/schema`.

## 1. Exemplo de `raw_projects`

Tabela bronze com payload original do GitLab.

```json
{
  "id": 10,
  "name": "API Core",
  "path": "backend/api-core",
  "last_synced_at": "2026-02-19T14:00:00Z",
  "raw_metadata": {
    "id": 10,
    "name": "API Core",
    "path_with_namespace": "backend/api-core",
    "visibility": "private",
    "default_branch": "main",
    "web_url": "https://gitlab.example.com/backend/api-core"
  }
}
```

Leitura importante:
- `raw_metadata` guarda o payload completo e nao deve ser retornado diretamente pela API final.

## 2. Exemplo de `issues`

Tabela silver normalizada com `jsonb` leve para labels e assignees.

```json
{
  "id": 215,
  "gitlab_issue_id": 887766,
  "project_id": 10,
  "iid": 1024,
  "title": "Corrigir timeout no gateway",
  "current_canonical_state": "IN_PROGRESS",
  "metadata_labels": ["Bug", "Incidente_Prod"],
  "assignees": ["joao.silva", "maria.santos"],
  "gitlab_created_at": "2026-02-17T08:00:00Z"
}
```

Leituras importantes:
- `metadata_labels` representa classificacoes de negocio.
- `assignees` e um array JSONB simples, facil de desserializar na API.

## 3. Exemplo de `issue_events`

Tabela silver de eventos tratados.

```json
{
  "id": 9001,
  "gitlab_event_id": 450001,
  "issue_id": 215,
  "project_id": 10,
  "issue_iid": 1024,
  "author_name": "joao.silva",
  "raw_label_added": "Doing",
  "raw_label_removed": "Backlog",
  "mapped_canonical_state": "IN_PROGRESS",
  "event_timestamp": "2026-02-18T10:00:00Z",
  "is_noise": false,
  "cycle_count": 0
}
```

Leituras importantes:
- `mapped_canonical_state` ja representa o estado apos o mapeamento de labels.
- `cycle_count` ajuda a detectar retrabalho.

## 4. Exemplo de `issue_comments`

Tabela silver com comentarios estruturados.

```json
{
  "id": 30001,
  "gitlab_note_id": 78001,
  "issue_id": 215,
  "author_name": "analista.qa",
  "body": "Estourou erro 500 na homologacao. Retornando para dev.",
  "comment_timestamp": "2026-02-19T15:30:00Z"
}
```

## 5. Exemplo de `raw_events.raw_payload`

Exemplo resumido de um payload bruto em JSONB.

```json
{
  "id": 450001,
  "user": {
    "username": "joao.silva"
  },
  "created_at": "2026-02-18T10:00:00Z",
  "resource_type": "Issue",
  "label": {
    "name": "Doing"
  },
  "action": "add"
}
```

Leitura importante:
- o bronze preserva o payload bruto; o silver extrai apenas o que interessa para a API e para metricas.

## 6. Exemplo de `vw_issue_lifecycle_metrics`

View gold com metricas consolidadas por issue.

```json
{
  "issue_id": 215,
  "project_id": 10,
  "project_path": "backend/api-core",
  "issue_iid": 1024,
  "gitlab_issue_id": 887766,
  "issue_title": "Corrigir timeout no gateway",
  "current_canonical_state": "DONE",
  "metadata_labels": ["Bug", "Incidente_Prod"],
  "assignees": ["joao.silva"],
  "gitlab_created_at": "2026-02-17T08:00:00Z",
  "first_in_progress_at": "2026-02-18T10:00:00Z",
  "first_done_at": "2026-02-21T16:40:00Z",
  "is_completed": true,
  "qa_to_dev_return_count": 1,
  "blocked_time_hours": 8.5,
  "cycle_time_hours": 100.8,
  "lead_time_hours": 128.7,
  "backlog_wait_hours": 26.0,
  "flow_efficiency_pct": 78.3
}
```

Leitura importante:
- essa view ja entrega quase tudo o que os endpoints agregados precisam.

## 7. Exemplo de `vw_projects_catalog`

View gold orientada a catalogo para filtros e seletores.

```json
{
  "id": 10,
  "name": "API Core",
  "path": "backend/api-core",
  "group_path": "backend",
  "total_issues": 1204,
  "last_synced_at": "2026-02-19T14:00:00Z"
}
```

Leitura importante:
- `group_path` e derivado removendo o ultimo segmento do `path`.

## 8. Como os JSONB viram resposta da API

### `issues.metadata_labels`

No banco:

```json
["Bug", "Incidente_Prod"]
```

Na API:

```json
{
  "metadata_labels": ["Bug", "Incidente_Prod"]
}
```

### `issues.assignees`

No banco:

```json
["joao.silva", "maria.santos"]
```

Na API:

```json
{
  "assignees": ["joao.silva", "maria.santos"]
}
```

### Timeline derivada

Eventos e comentarios distintos no banco podem ser combinados pela API em uma unica linha do tempo:

```json
[
  {
    "type": "state_change",
    "timestamp": "2026-02-18T10:00:00Z",
    "actor": "joao.silva",
    "from_state": "BACKLOG",
    "to_state": "IN_PROGRESS",
    "duration_in_previous_state_mins": 2880
  },
  {
    "type": "comment",
    "timestamp": "2026-02-19T15:30:00Z",
    "actor": "analista.qa",
    "body": "Estourou erro 500 na homologacao. Retornando para dev."
  }
]
```

## 9. Exemplo de leitura por endpoint

### `/api/v1/projects`

Fonte principal:

```json
{
  "id": 10,
  "name": "API Core",
  "path": "backend/api-core",
  "total_issues": 1204,
  "last_synced_at": "2026-02-19T14:00:00Z"
}
```

### `/api/v1/issues/215/timeline`

Composicao:
- `issues` para cabecalho da issue
- `issue_events` ou `vw_issue_state_transitions` para transicoes
- `issue_comments` para comentarios

### `/api/v1/metrics/quality`

Fonte principal:
- `vw_issue_lifecycle_metrics`

Indicadores derivados:
- `ping_pong_rate_pct`
- `bypass_rate_pct`
- `total_blocked_time_hours`
- `bug_ratio_pct`

## 10. Observacoes finais

- Os exemplos acima nao sao dumps reais; sao modelos representativos para documentacao.
- Se o formato de `assignees` ou `metadata_labels` mudar no processo de transformacao, este documento deve ser atualizado junto com os SQLs e contratos da API.

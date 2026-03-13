# Database Documentation

## Overview

Este projeto usa um modelo em camadas para separar dados brutos do GitLab, dados tratados e views analiticas prontas para consumo pela API.

- **Bronze**: payloads brutos preservados, com foco em rastreabilidade e reprocessamento.
- **Silver**: entidades normalizadas e eventos ja tratados para consulta operacional.
- **Gold**: views analiticas com agregacoes, calculos de ciclo e metricas de engenharia.
- **Config**: tabelas auxiliares de mapeamento e auditoria do processo de transformacao.

Os arquivos SQL que definem o schema ficam em `db/schema`.

## Layers

### Bronze

Objetivo: guardar payloads e eventos como vieram do GitLab, minimizando perda de contexto.

- `raw_projects`
- `raw_events`
- `raw_issues`

### Silver

Objetivo: expor entidades e eventos limpos, com estados canonicamente mapeados e estrutura pronta para consulta detalhada.

- `projects`
- `issues`
- `issue_events`
- `issue_comments`

### Gold

Objetivo: concentrar calculos pesados e fornecer bases consistentes para metricas agregadas.

- `vw_issue_state_transitions`
- `vw_issue_state_intervals`
- `vw_issue_lifecycle_metrics`
- `vw_project_engineering_metrics`
- `vw_projects_catalog`

### Config

Objetivo: suportar o processo de tratamento, mapeamento e auditoria.

- `sync_state`
- `state_mapping`
- `metadata_mapping`
- `unknown_labels_log`
- `dead_letter_queue`

## Bronze Tables

### `raw_projects`

Fonte: `db/schema/000001_create_raw_projects_table.up.sql`

Objetivo: armazenar o payload completo de projetos vindo do GitLab.

Colunas principais:
- `id`: ID do projeto no GitLab.
- `name`: nome do projeto.
- `path`: path completo do projeto.
- `last_synced_at`: timestamp do ultimo sync conhecido.
- `raw_metadata` (`jsonb`): payload integral retornado pela API do GitLab.

Observacoes:
- e uma tabela bronze, com foco em preservacao do payload bruto;
- serve de base para a tabela silver `projects`.

### `raw_events`

Fonte: `db/schema/000002_create_raw_events_table.up.sql`

Objetivo: armazenar eventos brutos de issue extraidos da API do GitLab.

Colunas principais:
- `gitlab_event_id`: ID do evento no GitLab, quando disponivel.
- `project_id`: projeto do evento.
- `issue_iid`: iid da issue dentro do projeto.
- `event_type`: tipo do evento, por exemplo `label_event`, `note`, `issue_update`.
- `raw_payload` (`jsonb`): payload completo do evento.
- `processed`: indica se o evento ja foi transformado para silver.

Observacoes:
- e append-only por natureza;
- alimenta `issue_events` e `issue_comments`.

### `raw_issues`

Fonte: `db/schema/000012_create_raw_issues_table.up.sql`

Objetivo: preservar metadados brutos de issues, incluindo titulo, descricao e payload completo.

Colunas principais:
- `gitlab_issue_id`: ID global da issue no GitLab.
- `project_id`: projeto da issue.
- `iid`: numero visivel da issue dentro do projeto.
- `title`: titulo da issue.
- `description`: descricao textual.
- `state`: estado bruto do GitLab, por exemplo `opened` ou `closed`.
- `raw_payload` (`jsonb`): payload completo da issue.

## Silver Tables

### `projects`

Fonte: `db/schema/000004_create_projects_table.up.sql`

Objetivo: representar projetos de forma normalizada para analise e consumo.

Colunas principais:
- `id`: ID do projeto no GitLab.
- `name`: nome do projeto.
- `path`: path do projeto.
- `last_synced_at`: ultimo sync consolidado.

Relacionamentos:
- e referenciada por `issues` e `issue_events`.

### `issues`

Fonte: `db/schema/000005_create_issues_table.up.sql`

Objetivo: armazenar issues normalizadas, com metadados estruturados e estado canonico atual.

Colunas principais:
- `id`: ID interno da API/banco.
- `gitlab_issue_id`: ID global da issue no GitLab.
- `project_id`: FK para `projects.id`.
- `iid`: numero visivel da issue no projeto.
- `title`: titulo da issue.
- `current_canonical_state`: cache do ultimo estado canonico conhecido.
- `metadata_labels` (`jsonb`): labels categoricas nao relacionadas a estado.
- `assignees` (`jsonb`): lista de usuarios associados a issue.
- `gitlab_created_at`: data original de criacao da issue.

Observacoes:
- `metadata_labels` e `assignees` sao arrays em JSONB;
- esta tabela e base para timeline, listagem detalhada e metricas derivadas.

### `issue_events`

Fonte: `db/schema/000006_create_issue_events_table.up.sql`

Objetivo: guardar eventos de mudanca de estado ja normalizados e mapeados para estados canonicos.

Colunas principais:
- `gitlab_event_id`: ID do evento no GitLab.
- `issue_id`: FK para `issues.id`.
- `project_id`: FK para `projects.id`.
- `issue_iid`: iid da issue.
- `author_name`: autor do evento.
- `raw_label_added`: label adicionada no evento bruto.
- `raw_label_removed`: label removida no evento bruto.
- `mapped_canonical_state`: estado canonico resultante.
- `event_timestamp`: momento do evento.
- `is_noise`: indica transicao considerada ruido.
- `cycle_count`: contador de ciclos de retrabalho.

Observacoes:
- `is_noise = true` representa eventos ignoraveis para metricas, como transicoes muito rapidas;
- `cycle_count` ajuda a medir ping-pong e retrabalho.

### `issue_comments`

Fonte: `db/schema/000007_create_issue_comments_table.up.sql`

Objetivo: guardar comentarios estruturados de issues.

Colunas principais:
- `gitlab_note_id`: ID do comentario no GitLab.
- `issue_id`: FK para `issues.id`.
- `author_name`: autor do comentario.
- `body`: texto do comentario.
- `comment_timestamp`: momento do comentario.

Observacoes:
- usada principalmente para montar a timeline detalhada da issue.

## Gold Views

### `vw_issue_state_transitions`

Fonte: `db/schema/000013_create_golden_engineering_views.up.sql`

Objetivo: gerar uma timeline limpa de transicoes por issue, removendo ruido e estados consecutivos duplicados.

Principais campos:
- `issue_id`, `project_id`, `project_path`, `issue_iid`
- `previous_canonical_state`, `canonical_state`, `next_canonical_state`
- `entered_at`, `exited_at`
- `duration_hours_to_next_state`
- `cycle_count`

Uso na API:
- pode suportar o endpoint `/issues/{id}/timeline`.

### `vw_issue_state_intervals`

Objetivo: representar intervalos de permanencia por estado, incluindo duracao em horas e se o intervalo ainda esta aberto.

Principais campos:
- `canonical_state`
- `entered_at`
- `exited_at`
- `is_open_interval`
- `duration_hours`

Uso na API:
- apoio para calculos de aging, blocked time e tempo em estado.

### `vw_issue_lifecycle_metrics`

Objetivo: consolidar metricas de ciclo por issue, incluindo lead time, cycle time, blocked time, backlog wait, retrabalho e eficiencia de fluxo.

Principais campos:
- `current_canonical_state`
- `is_completed`
- `skipped_in_progress_flag`
- `qa_to_dev_return_count`
- `blocked_time_hours`
- `cycle_time_hours`
- `lead_time_hours`
- `backlog_wait_hours`
- `flow_efficiency_pct`
- `first_done_at`, `final_done_at`

Uso na API:
- principal fonte para:
  - `/api/v1/metrics/delivery`
  - `/api/v1/metrics/quality`
  - `/api/v1/metrics/wip`
  - `/api/v1/issues`

### `vw_project_engineering_metrics`

Objetivo: resumir metricas por projeto, incluindo throughput, tempos medios, retrabalho, ghost work e bloqueios.

Principais campos:
- `total_issues`
- `completed_issues`
- `in_progress_issues`
- `qa_review_issues`
- `blocked_issues`
- `avg_lead_time_hours`
- `avg_cycle_time_hours`
- `ghost_work_pct`
- `rework_issue_pct`
- `blocked_issue_pct`

Uso na API:
- apoio a visoes por projeto e catalogos enriquecidos.

### `vw_projects_catalog`

Objetivo: oferecer catalogo de projetos pronto para seletores e APIs, incluindo `group_path`, total de issues e ultimo sync consolidado.

Principais campos:
- `id`
- `name`
- `path`
- `group_path`
- `total_issues`
- `last_synced_at`

Uso na API:
- principal fonte para `/api/v1/projects`;
- base para agregacao de `/api/v1/groups`.

## Config Tables

### `sync_state`

Objetivo: controlar o cursor de sincronizacao incremental por projeto.

Campos principais:
- `project_id`
- `last_synced_at`

### `state_mapping`

Objetivo: mapear labels do GitLab para estados canonicos da plataforma.

Campos principais:
- `gitlab_label_name`
- `canonical_state`
- `description`

Estados canonicos permitidos:
- `BACKLOG`
- `IN_PROGRESS`
- `QA_REVIEW`
- `BLOCKED`
- `DONE`
- `CANCELED`
- `UNKNOWN`

### `metadata_mapping`

Objetivo: mapear labels nao relacionadas a estado para chaves categoricas de negocio.

Campos principais:
- `gitlab_label_name`
- `metadata_key`

### `unknown_labels_log`

Objetivo: registrar labels encontradas que ainda nao foram mapeadas.

Campos principais:
- `label_name`
- `occurrence_count`
- `first_seen_at`
- `last_seen_at`

### `dead_letter_queue`

Objetivo: registrar eventos com erro de processamento, parse ou persistencia.

Campos principais:
- `project_id`
- `issue_iid`
- `event_type`
- `raw_payload` (`jsonb`)
- `error_message`
- `error_category`
- `retry_count`
- `max_retries`
- `resolved`

## JSONB Fields

Campos `jsonb` relevantes neste projeto:

- `raw_projects.raw_metadata`
- `raw_events.raw_payload`
- `raw_issues.raw_payload`
- `issues.metadata_labels`
- `issues.assignees`
- `dead_letter_queue.raw_payload`

Leituras importantes:
- `issues.metadata_labels` representa um array de labels de negocio, nao de transicao de estado.
- `issues.assignees` representa um array simples de usernames ou identificadores equivalentes.
- payloads brutos em bronze existem para rastreabilidade e reprocessamento, nao para consumo direto da API.

## Relationships

Relacoes principais:

- `projects.id` -> `issues.project_id`
- `projects.id` -> `issue_events.project_id`
- `issues.id` -> `issue_events.issue_id`
- `issues.id` -> `issue_comments.issue_id`
- `raw_projects.id` -> `sync_state.project_id`

Fluxo logico de dados:

1. GitLab alimenta tabelas bronze.
2. Processamento transforma bronze em silver.
3. Views gold consolidam metricas.
4. A API le silver para detalhe e gold para agregacao.

## API Consumption Map

- `/api/v1/projects` -> `vw_projects_catalog`
- `/api/v1/groups` -> agregacao sobre `vw_projects_catalog`
- `/api/v1/users` -> `issues` + `vw_issue_lifecycle_metrics`
- `/api/v1/users/{username}/performance` -> `issues` + `vw_issue_lifecycle_metrics`
- `/api/v1/metrics/delivery` -> `vw_issue_lifecycle_metrics`
- `/api/v1/metrics/quality` -> `vw_issue_lifecycle_metrics`
- `/api/v1/metrics/wip` -> `vw_issue_lifecycle_metrics`
- `/api/v1/issues` -> `vw_issue_lifecycle_metrics`
- `/api/v1/issues/{id}/timeline` -> `issues` + `issue_events` + `issue_comments` ou `vw_issue_state_transitions`

## Notes

- `group_path` e derivado a partir de `projects.path` na view `vw_projects_catalog`.
- `current_canonical_state` e um cache util para leitura rapida, mas as views gold refinam esse estado a partir das transicoes.
- `ghost work` e `rework` nao existem como coluna bruta unica; sao derivados das regras nas views gold.
- Para exemplos de registros e campos `jsonb`, veja `docs/database/examples.md`.

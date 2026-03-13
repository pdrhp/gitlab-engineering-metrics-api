# GitLab Engineering Metrics API Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Construir a primeira versao da API read-only de metricas de engenharia, com autenticacao por cliente, endpoints versionados e contratos alinhados ao PRD.

**Architecture:** A API sera organizada em camadas simples de handler, service e repository. O acesso ao banco sera feito com SQL explicito sobre tabelas silver e views gold, mantendo regras HTTP, validacoes e queries separadas para facilitar manutencao.

**Tech Stack:** Go, net/http ou chi, Postgres, database/sql ou sqlx, testes com Go testing.

---

### Task 1: Bootstrap do projeto

**Files:**
- Create: `go.mod`
- Create: `cmd/api/main.go`
- Create: `internal/app/app.go`
- Create: `internal/app/routes.go`

**Step 1: Inicializar modulo Go**

Criar `go.mod` com o nome do modulo e dependencias minimas.

**Step 2: Criar entrypoint HTTP**

Criar `cmd/api/main.go` com bootstrap de config, banco e servidor.

**Step 3: Criar wiring da aplicacao**

Criar `internal/app/app.go` e `internal/app/routes.go` para montar middlewares e rotas.

**Step 4: Validar build inicial**

Run: `go test ./...`
Expected: compila ou falha apenas por arquivos ainda nao criados nas tasks seguintes.

### Task 2: Configuracao e conexao com banco

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/database/postgres.go`

**Step 1: Modelar configuracao**

Adicionar leitura de host, port, dbname, user, password, sslmode e credenciais de clientes.

**Step 2: Implementar conexao Postgres**

Criar pool com limites e ping inicial.

**Step 3: Testar carga de configuracao**

Run: `go test ./...`
Expected: testes de config passam.

### Task 3: Middleware e respostas padrao

**Files:**
- Create: `internal/http/middleware/auth.go`
- Create: `internal/http/middleware/logging.go`
- Create: `internal/http/middleware/recovery.go`
- Create: `internal/http/middleware/request_id.go`
- Create: `internal/http/responses/error.go`
- Create: `internal/auth/credentials.go`

**Step 1: Implementar parser de credenciais**

Ler env e manter mapa simples de `client_id -> client_secret`.

**Step 2: Implementar middleware de auth**

Validar headers e rejeitar requests invalidas com `401`.

**Step 3: Implementar middlewares basicos**

Adicionar request id, recovery e logging estruturado.

**Step 4: Padronizar payload de erro**

Criar helper para erros `400`, `401`, `404`, `422` e `500`.

### Task 4: Modelos de dominio e contratos

**Files:**
- Create: `internal/domain/project.go`
- Create: `internal/domain/group.go`
- Create: `internal/domain/user.go`
- Create: `internal/domain/metrics.go`
- Create: `internal/domain/issue.go`
- Create: `internal/domain/timeline.go`

**Step 1: Definir structs de resposta**

Modelar payloads alinhados ao `docs/openapi.yaml`.

**Step 2: Definir filtros de entrada**

Criar structs para filtros de catalogo, metricas e issues.

### Task 5: Catalog endpoints

**Files:**
- Create: `internal/repositories/projects_repository.go`
- Create: `internal/repositories/groups_repository.go`
- Create: `internal/repositories/users_repository.go`
- Create: `internal/services/catalog_service.go`
- Create: `internal/http/handlers/projects_handler.go`
- Create: `internal/http/handlers/groups_handler.go`
- Create: `internal/http/handlers/users_handler.go`

**Step 1: Implementar queries de catalogo**

Usar `vw_projects_catalog` e fontes derivadas para projetos, grupos e usuarios.

**Step 2: Implementar service de catalogo**

Aplicar validacoes de filtros e orquestrar repositories.

**Step 3: Implementar handlers**

Expor `/projects`, `/groups` e `/users`.

**Step 4: Testar endpoints**

Run: `go test ./...`
Expected: testes de handlers e services passam.

### Task 6: Metrics endpoints

**Files:**
- Create: `internal/repositories/metrics_repository.go`
- Create: `internal/services/metrics_service.go`
- Create: `internal/http/handlers/delivery_handler.go`
- Create: `internal/http/handlers/quality_handler.go`
- Create: `internal/http/handlers/wip_handler.go`

**Step 1: Implementar queries de metricas**

Usar `vw_issue_lifecycle_metrics` para delivery, quality e wip.

**Step 2: Implementar service de metricas**

Validar intervalo de datas, filtros e combinacoes invalidas.

**Step 3: Implementar handlers**

Expor `/metrics/delivery`, `/metrics/quality` e `/metrics/wip`.

**Step 4: Testar contratos**

Run: `go test ./...`
Expected: payloads e status codes corretos.

### Task 7: Issues endpoints

**Files:**
- Create: `internal/repositories/issues_repository.go`
- Create: `internal/repositories/timeline_repository.go`
- Create: `internal/services/issues_service.go`
- Create: `internal/http/handlers/issues_handler.go`
- Create: `internal/http/handlers/timeline_handler.go`

**Step 1: Implementar query de listagem de issues**

Usar `vw_issue_lifecycle_metrics` com filtros e paginacao.

**Step 2: Implementar query de timeline**

Combinar `issues`, `issue_events`, `issue_comments` e/ou `vw_issue_state_transitions`.

**Step 3: Implementar service e handlers**

Expor `/issues` e `/issues/:id/timeline`.

**Step 4: Testar cenarios de not found e filtros invalidos**

Run: `go test ./...`
Expected: `404`, `400` e `422` corretos.

### Task 8: Observabilidade e readiness

**Files:**
- Create: `internal/observability/logger.go`
- Create: `internal/observability/metrics.go`

**Step 1: Adicionar logs estruturados**

Registrar metodo, rota, status, latencia e request id.

**Step 2: Adicionar metricas basicas**

Contar requests, erros e latencia por endpoint.

### Task 9: Testes de integracao e reconciliacao

**Files:**
- Create: `internal/...` arquivos de teste conforme implementacao

**Step 1: Criar fixtures ou ambiente controlado**

Preparar base de testes com dados minimos representativos.

**Step 2: Validar respostas contra SQL de referencia**

Comparar resultados dos endpoints com queries homologadas.

**Step 3: Rodar suite completa**

Run: `go test ./...`
Expected: tudo verde.

### Task 10: Revisao final e documentacao operacional

**Files:**
- Modify: `docs/openapi.yaml`
- Modify: `docs/api-architecture.md`
- Modify: `docs/prd.md`

**Step 1: Atualizar docs conforme implementacao real**

Garantir que contratos e detalhes finais reflitam o codigo.

**Step 2: Validar prontidao**

Checar autenticacao, erros, observabilidade e latencia.

**Step 3: Preparar handoff**

Registrar como subir a API, configurar env e validar endpoints.

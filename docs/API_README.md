# GitLab Engineering Metrics API

API REST read-only em Go para exposicao de metricas de engenharia a partir do banco `gitlab_elt`.

## Quick Start

### Prerequisites

- Go 1.25+
- PostgreSQL (com banco `gitlab_elt` acessivel)
- Credenciais de acesso ao banco

### Build

```bash
go build -o api cmd/api/main.go
```

### Run

```bash
# Configuracao minima obrigatoria
export DB_USER=postgres
export DB_PASSWORD=your_password
export CLIENT_CREDENTIALS="myclient:mysecret"

# Opcional (valores default)
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=gitlab_metrics
export DB_SSLMODE=disable
export SERVER_ADDR=:8080
export LOG_FORMAT=json
export LOG_LEVEL=info

./api
```

A API estara disponivel em `http://localhost:8080`

## Endpoints

### Publicos (sem autenticacao)

| Metodo | Endpoint | Descricao |
|--------|----------|-----------|
| GET | `/health` | Health check do servico e banco |
| GET | `/metrics` | Metricas de performance da API |

### Protegidos (requer autenticacao)

| Metodo | Endpoint | Descricao |
|--------|----------|-----------|
| GET | `/api/v1/projects` | Lista projetos |
| GET | `/api/v1/groups` | Lista grupos |
| GET | `/api/v1/users` | Lista usuarios |
| GET | `/api/v1/metrics/delivery` | Metricas de delivery |
| GET | `/api/v1/metrics/quality` | Metricas de qualidade |
| GET | `/api/v1/metrics/wip` | Metricas de WIP |
| GET | `/api/v1/issues` | Lista issues |
| GET | `/api/v1/issues/:id/timeline` | Timeline de uma issue |

## Autenticacao

A API usa autenticacao via **Client Credentials** com headers HTTP:

```bash
curl -H "X-Client-ID: seu_client_id" \
     -H "X-Client-Secret: seu_client_secret" \
     http://localhost:8080/api/v1/projects
```

### Configurando Credenciais

Defina a variavel de ambiente `CLIENT_CREDENTIALS`:

```bash
# Formato: "client1:secret1,client2:secret2"
export CLIENT_CREDENTIALS="dashboard:abc123,mobile:xyz789,etl:def456"
```

Multiplos pares de credenciais sao suportados (separados por virgula).

## Environment Variables

| Variavel | Obrigatorio | Padrao | Descricao |
|----------|-------------|--------|-----------|
| `DB_USER` | Sim | - | Usuario do PostgreSQL |
| `DB_PASSWORD` | Sim | - | Senha do PostgreSQL |
| `CLIENT_CREDENTIALS` | Sim | - | Credenciais de client (formato: "client:secret,client:secret") |
| `DB_HOST` | Nao | localhost | Host do PostgreSQL |
| `DB_PORT` | Nao | 5432 | Porta do PostgreSQL |
| `DB_NAME` | Nao | gitlab_metrics | Nome do banco |
| `DB_SSLMODE` | Nao | disable | Modo SSL (disable/require/verify-ca/verify-full) |
| `SERVER_ADDR` | Nao | :8080 | Endereco do servidor |
| `MAX_OPEN_CONNS` | Nao | 25 | Maximo de conexoes abertas |
| `MAX_IDLE_CONNS` | Nao | 5 | Maximo de conexoes ociosas |
| `CONN_MAX_LIFETIME` | Nao | 5m | Tempo maximo de vida da conexao |
| `LOG_FORMAT` | Nao | json | Formato de log (json ou text) |
| `LOG_LEVEL` | Nao | info | Nivel de log (debug/info/warn/error) |

## Exemplos de Uso

### Health Check

```bash
curl http://localhost:8080/health
# Response: {"status":"healthy"}
```

### Listar Projetos

```bash
curl -H "X-Client-ID: client1" \
     -H "X-Client-Secret: secret1" \
     http://localhost:8080/api/v1/projects

# Com filtros
curl -H "X-Client-ID: client1" \
     -H "X-Client-Secret: secret1" \
     "http://localhost:8080/api/v1/projects?group_path=my-group&search=api"
```

### Metricas de Delivery

```bash
curl -H "X-Client-ID: client1" \
     -H "X-Client-Secret: secret1" \
     "http://localhost:8080/api/v1/metrics/delivery?start_date=2026-01-01&end_date=2026-01-31"
```

### Listar Issues

```bash
curl -H "X-Client-ID: client1" \
     -H "X-Client-Secret: secret1" \
     "http://localhost:8080/api/v1/issues?page=1&page_size=25&state=opened"
```

### Timeline de Issue

```bash
curl -H "X-Client-ID: client1" \
     -H "X-Client-Secret: secret1" \
     http://localhost:8080/api/v1/issues/123/timeline
```

## Database Setup

A API espera as seguintes views e tabelas:

### Views Recomendadas (Gold)

- `vw_projects_catalog` - Catalogo de projetos
- `vw_issue_lifecycle_metrics` - Metricas de lifecycle de issues

### Tabelas Utilizadas

- `projects` - Metadados de projetos
- `issues` - Issues do GitLab
- `issue_events` - Eventos de issues
- `issue_comments` - Comentarios de issues

### Permissoes Necessarias

O usuario do banco deve ter permissao `SELECT` em todas as tabelas/views utilizadas.

## Observabilidade

### Health Check

```bash
curl http://localhost:8080/health
```

Retorna:
- `200 OK` - API e banco saudaveis
- `503 Service Unavailable` - Banco inacessivel

### Metrics

```bash
curl http://localhost:8080/metrics
```

Retorna metricas de:
- Total de requests
- Total de erros
- Latencia media por endpoint
- Latencia P95 por endpoint
- Contagem de status HTTP

### Logs

Logs estruturados em JSON (ou texto simples):

```json
{
  "time": "2026-03-09T15:30:00Z",
  "level": "INFO",
  "msg": "HTTP Request",
  "method": "GET",
  "path": "/api/v1/projects",
  "status": 200,
  "duration_ms": 45,
  "request_id": "abc-123-def"
}
```

### Request ID

Cada requisicao recebe um `X-Request-ID` automaticamente (ou usa o fornecido):

```bash
# Requisicao com request ID customizado
curl -H "X-Request-ID: my-trace-id" \
     -H "X-Client-ID: client1" \
     -H "X-Client-Secret: secret1" \
     http://localhost:8080/api/v1/projects
```

## Respostas de Erro

Todas as respostas de erro seguem o formato:

```json
{
  "code": "UNAUTHORIZED",
  "message": "Authentication required",
  "request_id": "abc-123-def"
}
```

### Codigos de Erro

| HTTP Status | Code | Descricao |
|-------------|------|-----------|
| 400 | `BAD_REQUEST` | Parametros invalidos |
| 401 | `UNAUTHORIZED` | Credenciais ausentes ou invalidas |
| 404 | `NOT_FOUND` | Recurso nao encontrado |
| 422 | `VALIDATION_ERROR` | Validacao semantica falhou |
| 500 | `INTERNAL_ERROR` | Erro interno do servidor |

## Desenvolvimento

### Estrutura do Projeto

```
cmd/api/main.go                      # Entry point
internal/
  app/routes.go                      # Rotas e wiring
  config/config.go                   # Configuracao
  http/middleware/                   # Middlewares (auth, logging, recovery, metrics, request_id)
  http/handlers/                     # HTTP handlers
  http/responses/                    # Respostas padronizadas
  domain/                            # Modelos de dominio
  services/                          # Logica de negocio
  repositories/                      # Queries SQL
  database/postgres.go               # Conexao com banco
  auth/credentials.go                # Validacao de credenciais
  observability/                     # Logger e metrics
docs/
  openapi.yaml                       # Especificacao OpenAPI
  prd.md                             # Requisitos do produto
  api-architecture.md                # Documentacao de arquitetura
test/integration/                    # Testes de integracao
```

### Testes

```bash
# Todos os testes
go test ./...

# Testes especificos
go test ./internal/http/handlers/...
go test ./internal/auth/...
```

### Build para Producao

```bash
# Build otimizado
CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o api cmd/api/main.go

# Docker (exemplo)
docker build -t gitlab-metrics-api .
```

## Troubleshooting

### Erro: "Failed to connect to database"

Verifique:
- Variaveis `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`
- Conectividade de rede com o banco
- Permissoes do usuario no banco

### Erro: "Authentication required"

Verifique:
- Headers `X-Client-ID` e `X-Client-Secret` estao presentes
- Valores correspondem aos configurados em `CLIENT_CREDENTIALS`

### Erro: "validation failed"

Verifique:
- Formato das datas: `YYYY-MM-DD`
- Range de datas nao excede 90 dias
- Valores obrigatorios estao presentes

### Performance Lenta

Verifique:
- Pool de conexoes: `MAX_OPEN_CONNS` (default: 25)
- Indices nas views gold do banco
- Uso de filtros para limitar resultados

## Documentacao Adicional

- [Especificacao OpenAPI](openapi.yaml) - Contrato completo da API
- [Requisitos do Produto](prd.md) - Contexto e requisitos
- [Arquitetura](api-architecture.md) - Detalhes da arquitetura

## Licenca

[Private - Uso interno]

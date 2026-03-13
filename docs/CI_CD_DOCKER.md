# CI/CD e Docker Hub

Este projeto possui pipeline no GitHub Actions para:

1. Rodar testes (`go test ./...`)
2. Validar build da API (`go build ./cmd/api`)
3. Gerar imagem Docker e publicar no Docker Hub

Arquivo do workflow: `.github/workflows/ci-cd.yml`

## Fluxo do pipeline

- Em `pull_request` para `main`:
  - Executa `test`
  - Executa `build`
- Em `push` na `main`:
  - Executa `test`
  - Executa `build`
  - Executa `docker` (build + push para Docker Hub)

## Configuracoes no GitHub (Repository Settings)

Configure no repositorio os seguintes valores.

### Secrets

- `DOCKERHUB_USERNAME`: seu usuario do Docker Hub
- `DOCKERHUB_TOKEN`: access token do Docker Hub (nao usar senha da conta)

### Variables

- `DOCKERHUB_IMAGE`: nome completo da imagem no Docker Hub

Exemplo:

`pedrohenrique/gitlab-engineering-metrics-api`

## Tags publicadas

O workflow publica as tags:

- `latest`
- `sha-<commit_sha_curto>`

Exemplo:

- `pedrohenrique/gitlab-engineering-metrics-api:latest`
- `pedrohenrique/gitlab-engineering-metrics-api:sha-a1b2c3d`

## Uso da imagem Docker

### Pull

```bash
docker pull pedrohenrique/gitlab-engineering-metrics-api:latest
```

Troque pelo valor que voce definiu em `DOCKERHUB_IMAGE`.

### Run

```bash
docker run --rm -p 8080:8080 \
  -e DB_HOST=host.docker.internal \
  -e DB_PORT=5432 \
  -e DB_NAME=gitlab_elt \
  -e DB_USER=postgres \
  -e DB_PASSWORD=gitlab_elt_dev \
  -e DB_SSLMODE=disable \
  -e SERVER_ADDR=:8080 \
  -e CLIENT_CREDENTIALS="myclient:mysecret" \
  pedrohenrique/gitlab-engineering-metrics-api:latest
```

## Variaveis de ambiente da aplicacao

As variaveis abaixo sao lidas pela aplicacao.

### Banco

- `DB_HOST` (default: `localhost`)
- `DB_PORT` (default: `5432`)
- `DB_NAME` (default: `gitlab_metrics`)
- `DB_USER` (default: vazio)
- `DB_PASSWORD` (default: vazio)
- `DB_SSLMODE` (default: `disable`)

### Servidor

- `SERVER_ADDR` (default: `:8080`)

### Pool de conexoes

- `MAX_OPEN_CONNS` (default: `25`)
- `MAX_IDLE_CONNS` (default: `5`)
- `CONN_MAX_LIFETIME` (default: `5m`)

### Autenticacao

- `CLIENT_CREDENTIALS`

Formato:

`client1:secret1,client2:secret2`

### Logs

- `LOG_FORMAT` (default: `json`)
- `LOG_LEVEL` (default: `info`)

> Observacao: o app tambem tenta carregar `.env` automaticamente na inicializacao.

## Build local da imagem

```bash
docker build -t gitlab-engineering-metrics-api:local .
docker run --rm -p 8080:8080 --env-file .env gitlab-engineering-metrics-api:local
```

## Troubleshooting rapido

- Push da imagem falhou com erro de login:
  - valide `DOCKERHUB_USERNAME` e `DOCKERHUB_TOKEN`
- Erro `name unknown` ao publicar:
  - confirme `DOCKERHUB_IMAGE` no formato `usuario/repositorio`
- API sobe, mas falha ao conectar no banco:
  - revise `DB_*` e conectividade da rede entre container e PostgreSQL

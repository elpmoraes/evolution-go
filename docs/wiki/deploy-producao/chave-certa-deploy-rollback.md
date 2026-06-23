# Chave Certa — deploy, rollback e transporte da build customizada

Este runbook descreve como operar a versão customizada do Evolution-Go usada pelo Chave Certa.

## Versões fixas

Evolution-Go:

```text
repo: git@github.com:elpmoraes/evolution-go.git
tag:  v0.7.1-cc.1
commit: b410b5a64391b23dd48e86d93f9dee132877e029
```

Whatsmeow:

```text
repo: git@github.com:elpmoraes/whatsmeow.git
tag:  v0.7.1-cc.1
commit: 8d941cdb210d47ca8332f66deffd4d5689bdf562
```

Imagem recomendada:

```text
ghcr.io/elpmoraes/evolution-go:0.7.1-cc.1
```

Se usar Docker Hub em vez de GHCR:

```text
elpmoraes/evolution-go:0.7.1-cc.1
```

## Variáveis recomendadas

```env
EVENT_IGNORE_GROUP=true
EVENT_IGNORE_STATUS=true
EVENT_IGNORE_NEWSLETTER=true
```

Essas flags reduzem trabalho inútil antes do decrypt. O webhook por organização deve continuar configurado normalmente, por exemplo:

```text
https://api.chave-certa.com/leads/webhooks/evolution-go/messages?organization_id=3&token=<token>
```

## Build da imagem

Em uma máquina com Docker e acesso GitHub:

```bash
git clone --recurse-submodules git@github.com:elpmoraes/evolution-go.git
cd evolution-go
git checkout v0.7.1-cc.1
git submodule update --init --recursive
docker build --build-arg VERSION=0.7.1-cc.1 -t ghcr.io/elpmoraes/evolution-go:0.7.1-cc.1 .
docker push ghcr.io/elpmoraes/evolution-go:0.7.1-cc.1
```

Se o submodule não vier no commit esperado, conferir:

```bash
git -C whatsmeow-lib rev-parse HEAD
```

O resultado deve ser:

```text
8d941cdb210d47ca8332f66deffd4d5689bdf562
```

## Deploy canário em uma VM existente

Antes de trocar a imagem, salvar o estado atual:

```bash
docker ps --filter name=evolution_go
docker inspect evolution_go --format '{{.Config.Image}}'
docker inspect evolution_go --format '{{json .Mounts}}'
docker inspect evolution_go --format '{{json .Config.Env}}' > /tmp/evolution-go-env-before.json
```

Atualizar a imagem no `docker-compose.yml` ou no comando de execução para:

```text
ghcr.io/elpmoraes/evolution-go:0.7.1-cc.1
```

Garantir as envs:

```env
EVENT_IGNORE_GROUP=true
EVENT_IGNORE_STATUS=true
EVENT_IGNORE_NEWSLETTER=true
```

Reiniciar:

```bash
docker compose pull evolution_go
docker compose up -d evolution_go
```

Se a VM não usa Compose, recriar o container preservando os mesmos volumes, rede, portas e envs do container atual.

## Validação após deploy

Validar:

```bash
docker logs --since=10m evolution_go
```

Critérios mínimos:

- container sobe sem QR novo obrigatório;
- instâncias continuam conectadas;
- webhook de mensagem privada chega ao Chave Certa;
- logs de `Node handling is taking long` para grupos/status/newsletter reduzem ou desaparecem;
- resposta inbound aparece no Chave Certa.

Teste funcional recomendado na `org-3`:

1. Enviar campanha para um lead real.
2. Responder pelo WhatsApp do lead.
3. Confirmar no Chave Certa:
   - `lead_messages` inbound criado;
   - `leads.has_responded=true`;
   - `lead_campaign_leads.responded_at` preenchido;
   - campanha finalizada não volta para `sending`;
   - resposta não cruza organização.

## Rollback para versão original

Rollback deve trocar apenas a imagem. Não apagar volumes.

Imagem original observada no ambiente:

```text
evoapicloud/evolution-go:latest
```

Procedimento com Compose:

```bash
docker compose stop evolution_go
```

Editar o `docker-compose.yml` e voltar a imagem para:

```text
evoapicloud/evolution-go:latest
```

Subir:

```bash
docker compose pull evolution_go
docker compose up -d evolution_go
```

Procedimento sem Compose:

1. Obter envs, portas, rede e volumes do container atual com `docker inspect`.
2. Parar/remover apenas o container.
3. Criar novo container com a imagem original e os mesmos volumes.

Exemplo conceitual:

```bash
docker stop evolution_go
docker rm evolution_go
docker run -d \
  --name evolution_go \
  --restart unless-stopped \
  --env-file /path/to/env \
  -v <volume-sessao>:/app/<path> \
  -p <porta>:<porta> \
  evoapicloud/evolution-go:latest
```

Nunca remover volumes durante rollback, porque eles carregam sessão, banco local e estado operacional.

## Transporte para outra VM

Opção A — usando imagem publicada:

```bash
docker pull ghcr.io/elpmoraes/evolution-go:0.7.1-cc.1
```

Copiar para a nova VM:

- `docker-compose.yml` ou comando `docker run`;
- arquivo `.env`;
- volumes, se a intenção for migrar sessão existente;
- URLs de webhook com `organization_id` e token corretos.

Subir:

```bash
docker compose up -d evolution_go
```

Opção B — sem registry, usando arquivo tar:

```bash
docker save ghcr.io/elpmoraes/evolution-go:0.7.1-cc.1 -o evolution-go-0.7.1-cc.1.tar
scp evolution-go-0.7.1-cc.1.tar usuario@nova-vm:/tmp/
ssh usuario@nova-vm 'docker load -i /tmp/evolution-go-0.7.1-cc.1.tar'
```

Depois usar a mesma tag no Compose da nova VM:

```text
ghcr.io/elpmoraes/evolution-go:0.7.1-cc.1
```

## Auditoria rápida

Conferir versão do código:

```bash
git ls-remote --tags git@github.com:elpmoraes/evolution-go.git v0.7.1-cc.1
git ls-remote --tags git@github.com:elpmoraes/whatsmeow.git v0.7.1-cc.1
```

Conferir imagem em execução:

```bash
docker inspect evolution_go --format '{{.Config.Image}}'
```

Conferir envs ativas:

```bash
docker inspect evolution_go --format '{{range .Config.Env}}{{println .}}{{end}}' | grep EVENT_IGNORE
```

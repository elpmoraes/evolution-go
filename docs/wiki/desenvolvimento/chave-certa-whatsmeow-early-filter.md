# Chave Certa — filtro antecipado no whatsmeow

## Objetivo

Reduzir a latência de mensagens privadas recebidas quando a sessão WhatsApp recebe volume alto de mensagens irrelevantes para o Chave Certa, principalmente grupos, status/stories, newsletters/canais e histórico.

O problema observado não estava no webhook do Chave Certa. O webhook era chamado e respondia 204 quando o Evolution-Go finalmente emitia o evento privado. O atraso acontecia antes disso, na fila serial de processamento do `whatsmeow`.

## Motivação técnica

O Evolution-Go já tinha filtros como `IgnoreGroups`, mas eles eram aplicados tarde demais:

1. WhatsApp entrega um nó de mensagem.
2. `whatsmeow` coloca o nó na fila interna.
3. `whatsmeow` parseia, descriptografa e processa a mensagem.
4. `whatsmeow` emite `events.Message`.
5. Evolution-Go recebe o evento.
6. Evolution-Go verifica `IgnoreGroups` e descarta.

Nesse fluxo, uma mensagem de grupo ignorada ainda podia consumir tempo de descriptografia/processamento antes de ser descartada. Em cenários com muitos grupos, isso atrasava mensagens privadas de leads.

## Alterações no fork whatsmeow

Repositório:

```text
git@github.com:elpmoraes/whatsmeow.git
```

Commit usado:

```text
8d941cd feat: add early incoming message filter
```

Tag:

```text
v0.7.1-cc.1
```

Arquivos alterados:

- `client.go`
- `message.go`

Foi adicionado o hook público:

```go
type IncomingMessageFilterDecision struct {
    Process bool
    Reason  string
}

type IncomingMessageFilter func(info *types.MessageInfo, node *waBinary.Node) IncomingMessageFilterDecision
```

Esse hook é chamado em `handleEncryptedMessage` depois de `parseMessageInfo(node)` e antes do fluxo de descriptografia/eventos.

Se `Process == false`:

- o nó é reconhecido com ACK;
- o corpo não é descriptografado;
- `events.Message` não é emitido;
- nenhum webhook de mensagem é disparado pelo Evolution-Go.

Por padrão, se o hook não for configurado, o comportamento do `whatsmeow` continua igual ao original.

## Alterações no fork Evolution-Go

Repositório:

```text
git@github.com:elpmoraes/evolution-go.git
```

Commit:

```text
b410b5a feat: filter ignored whatsapp messages before decrypt
```

Tag:

```text
v0.7.1-cc.1
```

Arquivos alterados:

- `.gitmodules`
- `pkg/config/env/env.go`
- `pkg/config/config.go`
- `pkg/whatsmeow/service/whatsmeow.go`
- `whatsmeow-lib` submodule

O Evolution-Go configura o hook no momento em que cria o client:

```go
client.IncomingMessageFilter = w.buildIncomingMessageFilter(cd.Instance)
```

Regras implementadas:

- `IgnoreGroups` da instância ou `EVENT_IGNORE_GROUP=true`: descarta grupos (`g.us`) antes do decrypt.
- `EVENT_IGNORE_STATUS=true`: descarta status/stories antes do decrypt.
- `EVENT_IGNORE_NEWSLETTER=true`: descarta newsletters/canais antes do decrypt.

## Por que o Evolution-Go também foi alterado

A mudança principal está no `whatsmeow`, mas o Evolution-Go precisou de uma integração mínima por dois motivos:

1. O Evolution-Go usa `replace go.mau.fi/whatsmeow => ./whatsmeow-lib`, então precisa apontar o submodule para o fork/tag correto.
2. Quem conhece as flags e configurações de instância é o Evolution-Go, então ele precisa configurar o hook do `whatsmeow`.

Sem essa integração, o `whatsmeow` teria o hook, mas ele nunca seria ativado.

## Comportamento esperado

Com as flags ativas, mensagens de grupo/status/newsletter são descartadas antes do processamento caro. Mensagens privadas continuam seguindo o fluxo normal e devem chegar ao webhook do Chave Certa com menor latência.

Flags recomendadas para Chave Certa:

```env
EVENT_IGNORE_GROUP=true
EVENT_IGNORE_STATUS=true
EVENT_IGNORE_NEWSLETTER=true
```

Essas flags não substituem o isolamento por organização no webhook. Cada instância ainda deve continuar usando webhook com `organization_id` e token corretos.

## Cuidados

- Não usar `latest` para deploy operacional.
- Usar tag fixa da imagem Docker.
- Manter o submodule em commit fixo.
- Não atualizar o `whatsmeow` sem testar, porque a API interna muda com frequência.
- O `whatsmeow/main` do fork não foi forçado: a versão operacional é a tag `v0.7.1-cc.1`.

# PRs

Como o agente trata review em PRs abertos deste repo.

Watch: em todo ciclo que tocar GitHub, listar PRs abertos e threads de review **não resolvidas**. Não esperar o autor pedir. Não mergear.

Classificação: usar o badge do comentário (`P1` / `P2`) quando existir (Codex e similares). Sem badge, classificar pelo efeito:

| | P1 | P2 |
|---|----|----|
| Efeito | bug, leak, deadline mentiroso, dado destruído, CI vermelho, contrato do spec quebrado | nit, estilo, robustez extra, teste opcional, sugestão fora do corte do PR |
| Ação | **resolver no código** | **não implementar** neste PR |

## P1 — resolver

1. Verificar se o achado é real neste código (não concordar por performance).
2. Teste que falha no comportamento citado, depois o fix mínimo.
3. Push na branch do PR. Responder **no thread** com o SHA e o que mudou.
4. Marcar o thread resolvido depois da resposta.

Não responder “vou corrigir” sem o commit já no remoto.

## P2 — responder, não implementar

Responder **no thread** (não num comentário solto no PR) com:

- **Risco:** baixo / médio / alto, e *o que quebra* se ignorarmos (uma frase).
- **Edge cases:** 2–4 situações concretas (não uma lista genérica).
- **Por que fica fora deste PR:** corte do spec, YAGNI, ou já coberto por teste/caminho existente.

Não fechar o thread. O reviewer decide se promove a P1.

Exemplo de forma:

```
P2 — não implementando neste PR.

Risco: baixo. Se apktool for um script que faz exec do java, o Cancel já mata o grupo; o caso residual é filho que dá setsid.

Edge cases:
- wrapper `apktool` → java que herda stdout (já P1)
- processo que chama `setsid` / daemonize (grupo novo; fora)
- Windows (Setpgid não aplica; Job Object é outro PR)

Fora deste PR: spec execx é Unix/adb no MVP; Job Object / setsid é v1 se um caller real precisar.
```

## Ordem

P1 primeiro (código + teste + push + reply). P2 depois, só reply. Cap de 3 fixes de código por ciclo (P1). Reply de P2 não conta no cap.

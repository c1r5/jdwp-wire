# development-rules

Regras de repo e fluxo pra `jdwp-wire`. Código mente, este arquivo acompanha no mesmo PR.

Stack: **Go 1.23+**. Binário `jdt`. Sem Python no core. JVM só como ferramenta externa (`apktool`, `apksigner`) ou, no v1, módulo `watch` separado.

Ler `AGENTS.md` antes de abrir issue.

---

## Forma do repo

```
jdwp-wire/
  AGENTS.md
  .grok/rules/development-rules.md
  README.md
  go.mod
  go.sum
  cmd/
    jdt/                 # main, só wiring de flags → módulos
  internal/
    cli/                 # flags, output humano/--json, exit codes
    execx/               # wrapper de subprocess (adb, apktool, android)
    device/
    apk/
    patch/
    jdwp/
    project/
    targets/
    androidcli/          # detect + shell-out da Android CLI oficial
    workspace/           # ./.jdt/<package>/
  testdata/              # APKs minúsculos, manifests, fixtures
  scripts/               # helpers locais (worktree, stack), não é produto
  watch/                 # v1 Kotlin/JDI — fora do go.mod até existir
```

Regras de forma:

- `cmd/jdt` não contém regra de negócio.
- Pacote novo só em `internal/<modulo>`. API pública do módulo = o que `cmd` e outros `internal` importam.
- Dependência entre módulos é acíclica. Sentido permitido: `execx` ← `device`/`apk`/`androidcli` ← `jdwp`/`patch`/`project`/`targets` ← `cli` ← `cmd`.
- Sem estado global de ADB (lição do repo antigo).
- Sem `init()`. Sem `package main` fora de `cmd/`.
- `internal/execx` é o único lugar que fala com o OS pra binário externo.

---

## Antes de código: issue

Documentação, planejamento, spec, PRD, desenho de módulo: **issue no GitHub via `gh` antes de qualquer implementação.**

```bash
gh issue create --title "spec(device): contrato do módulo" --body-file spec.md
```

- Uma issue por unidade revisável (módulo, comando, decisão).
- Branch e commits citam `#N`.
- Spec vive no body da issue (ou gist/arquivo linkado). Não começar feature “no feeling”.
- Mudança de `AGENTS.md` que altera corte MVP/v1 também ganha issue.

Exceção: typo / fix óbvio de build na `main` pode ir direto, PR mínimo, sem spec.

---

## Branches

| Tipo | Nome | Base |
|------|------|------|
| módulo novo | `feat/<modulo>` | `main` ou a dependência A |
| step de módulo grande | `feat/<modulo>-stepNN` | step anterior (stacked) |
| fix | `fix/<modulo>-<assunto>` | `main` ou a feat que quebrou |
| chore/docs | `chore/...` `docs/...` | `main` |
| spike | `spike/<assunto>` | `main`, não mergeia sem reescrever |

Exemplos: `feat/device`, `feat/device-step01`, `feat/jdwp` (base `feat/device`).

### Dependência entre módulos

Se B precisa de A:

1. `feat/A` a partir de `main`
2. `feat/B` a partir de `feat/A`
3. subir com **`gh stack`** (não PR isolado de B apontando pra `main`)

Não mergear B na `main` antes de A.

### Steps (módulo grande)

Criação de módulo = mudança grande. Quebrar em steps stacked:

```
feat/device              # opcional: guarda-chuva, ou só os steps
  feat/device-step01     # tipos + interface + fake
  feat/device-step02     # implementação adb
  feat/device-step03     # cmd jdt devices + testes integração
```

- Cada step é uma subbranch stacked em cima da anterior.
- **Se step N e N+1 juntos dão ~200 linhas ou menos, junte.** A regra existe pra revisão, não pra teatro de branch.
- Step demais com diff de 40 linhas é ruído. Step único de 800 linhas também. Alvo de revisão: 200–400 linhas úteis por PR da stack.

### Paralelo

Features **independentes** (sem import cruzado): worktrees, não stash-hopping.

```bash
git worktree add ../jdwp-wire-targets feat/targets
git worktree add ../jdwp-wire-project feat/project
```

- Um worktree por feat.
- Path: irmão do repo, `jdwp-wire-<branch-slug>`.
- Não reutilizar worktree pra outra feat sem `git worktree remove`.

---

## Commits

Histórico **linear**. Sem merge commit na `main`. Stack rebaseia; `main` só fast-forward / squash-merge da base da stack (um PR da vez, na ordem).

Cada commit:

- independente
- testa (`go test ./...` passa nele, não só no tip)
- faz uma coisa
- pode ser revertido sozinho sem deixar o tree quebrado

### Mensagem

```
feat (device): listar seriais via adb

- #12
- parse de `adb devices -l` (serial, estado, usb/product)
- erro explícito se 0 devices ou N sem -s
- fake em testdata pra não exigir hardware no unit
- fix: tratar `unauthorized` como estado, não como device usable
```

Formato da primeira linha:

```
<tipo> (<modulo>): <titulo_curto>
```

Tipos: `feat` `fix` `refactor` `test` `docs` `chore`.

Corpo = bullet points do que entrou **incluindo fixes depois do commit original**. Não criar `fixup!` solto no histórico final.

### Amend e remoto

Bug descoberto no meio do step, e o commit ainda é o da ponta da branch:

```bash
git add -p
git commit --amend --no-edit   # ou reescrever bullets
```

Se o commit **já está no remoto**:

```bash
git push --force-with-lease
```

Nunca `--force`. Nunca amend de commit que já não é o tip — aí é `rebase -i` na stack e `gh stack push` / force-with-lease de cada branch da stack.

Não amend de commit que outro worktree/pessoa já usou como base sem avisar e rebasear os de cima.

---

## Stack (`gh stack`)

- Ordem de review = ordem de dependência.
- Cada PR da stack deve buildar e testar isolado (o commit range dele).
- Descrição do PR: o que *este* step adiciona, não o módulo inteiro. Link da issue + link do PR de baixo.
- Rebase da base: de baixo pra cima, depois push `--force-with-lease` em cada uma.
- Não empilhar feat independente “pra ir junto”. Isso é worktree + PR separado na `main`.

---

## Go

- `gofmt` + `goimports` no hook / CI. Diff de formatação não mistura com feat.
- `golangci-lint` no CI. Não desliga linter no pacote novo.
- Teste no mesmo pacote (`device/device_test.go`). Integração que precisa de `adb` real: build tag `//go:build integration`.
- Preferir table test. Fixture em `testdata/`.
- Erros: `fmt.Errorf("device: list: %w", err)`. Sem panic em caminho de CLI.
- `context.Context` no primeiro argumento de chamada que sai da máquina (adb, android CLI, jdb).
- Dependência nova: commit próprio `chore (mods): add <lib>` com `go get` + `go mod tidy`. Justificar na issue se não for stdlib.
- Sem generics de enfeite. Sem interface “pra testar” no produtor — interface no consumidor.
- Shell-out: timeout obrigatório. Sem `CombinedOutput` infinito.

---

## Review e tamanho

- PR > ~400 linhas úteis: quebrar step ou justificar na issue (“gerado”, “golden”, “xml gigante”).
- Reviewer lê o commit, não o PR squash mental. Por isso commit testável.
- Não misturar refactor cosmético com comportamento.
- Atualizar `AGENTS.md` no PR que muda corte MVP/v1 ou contrato de CLI.

---

## CI mínimo

Em todo PR / step:

1. `go test ./...`
2. `gofmt -l` limpo
3. `golangci-lint run`
4. `go build ./cmd/jdt`

`main` sempre verde. Não push direto na `main`.

---

## O que não vai pro git

- APK de terceiro / app de alvo real
- keystore de produção
- captura HAR com token vivo (anonimizar fixture)
- `.jdt/` local
- binário `jdt` compilado

`testdata/` só fixture mínimo próprio ou gerado.

---

## Regras extras (vale adotar)

1. **Issue → branch → PR.** Sem branch órfã. `git checkout -b` só com `#N` no primeiro commit.
2. **Contrato do módulo no step01.** Tipos + interface + teste do fake *antes* da implementação `adb`. Review valida o desenho barato.
3. **Fake primeiro, device depois.** Unit não depende de emulador. Integration tag à parte.
4. **Um comando CLI por PR de wiring.** Implementar `internal/device` não precisa já expor `jdt devices` no mesmo step se inflar o diff. Pode ser o último step.
5. **Exit codes estáveis.** `0` ok, `2` uso, `3` sem device, `4` ferramenta ausente (`apktool`/`adb`). Documentar na issue do `cli`.
6. **Output `--json` é contrato.** Mudou schema = bump documentado na issue, não “ajuste de print”.
7. **Detect Android CLI, não assumir.** `android -h` / versão; fallback `adb`. Nunca tratar o `android` antigo do SDK como a CLI 1.0.
8. **Sem log que parece produto.** `slog` com nível. Default humano numa linha por passo (`[skip] patch: already debuggable`).
9. **Proibido estado estático mutável** em wrapper de ferramenta. Cada chamada é `execx.Run(ctx, name, args...)`.
10. **Spike tem data de morte.** `spike/*` não recebe review de produto. Ou vira spec+feat ou morre.
11. **Não commitar arquivo gerado pelo Studio** (`.idea` de verdade). `project` *escreve* um xml mínimo em `.jdt/`, versionamos só o *template* em `internal/project/testdata`.
12. **Segredo no diff = stop.** Se um teste logar token, rewrite do commit antes do push.
13. **Owner do stack é quem rebaseia.** Se duas pessoas tocam a mesma stack, uma só faz o rebase.
14. **Worktree não compartilha `./.jdt`.** Workspace é por clone/worktree.
15. **PR template curto:** issue, o que o step faz, como testar (`go test`, comando manual), risco (repack/assinatura).
16. **Não wrappear apktool.** É binário externo. Se o decode falhar, erro com stderr resumido, não parser do mundo apktool.
17. **Compat de flag.** Flag nova é ok. Renomear/remover flag = issue + menção no AGENTS.md.
18. **Último commit da stack atualiza README só se o comando já existir no tip da `main` depois do merge.** Evita README mentindo no meio da stack.

---

## Sequência típica (módulo `device`)

```bash
gh issue create --title "spec(device): listar devices e pidof" --body-file docs/spec-device.md
# #12

git checkout main && git pull
git checkout -b feat/device-step01
# tipos, interface, fake, testes
git commit -m "feat (device): contrato List/Pidof + fake"

git checkout -b feat/device-step02
# implementação adb
git commit -m "feat (device): List via adb devices -l"

# se step02 ficou pequeno demais, squash no step01 e delete step02

gh stack push
# PRs stacked, cada um aponta #12
```

Fix depois do push no tip:

```bash
git commit --amend
git push --force-with-lease
```

---

## Quando esta regra falha

- Step de 30 linhas “pra cumprir cerimônia” → junte.
- Amend de commit do meio da stack sem rebase dos de cima → histórico mentiroso, `gh stack` quebra.
- Feature independente no mesmo stack “porque é rápido” → worktree.
- Código sem issue porque “é óbvio” → não tem contexto na review daqui a um mês.

# worktrees

Worktrees deste repo vivem **dentro do clone**, em `.worktrees/` na raiz.
Não criar checkout irmão (`../jdwp-wire-<slug>`).

## Layout

```
.worktrees/<tipo>/<slug>/
```

`<tipo>` é o prefixo da branch: `feat`, `docs`, `fix`, `refactor`, `chore`, `spike`.
`<slug>` é o resto da branch depois da barra (o nome da tarefa).

| Branch | Path |
|--------|------|
| `feat/targets` | `.worktrees/feat/targets` |
| `feat/device-step01` | `.worktrees/feat/device-step01` |
| `docs/worktree-layout` | `.worktrees/docs/worktree-layout` |
| `fix/device-pidof` | `.worktrees/fix/device-pidof` |
| `refactor/execx-timeout` | `.worktrees/refactor/execx-timeout` |
| `chore/mods-foo` | `.worktrees/chore/mods-foo` |
| `spike/android-cli` | `.worktrees/spike/android-cli` |

Branch sem barra: `.worktrees/<tipo>/<branch>` se o tipo for óbvio; senão `.worktrees/chore/<branch>`.

## Comando

```bash
mkdir -p .worktrees/feat
git worktree add .worktrees/feat/targets -b feat/targets
# branch já existe:
git worktree add .worktrees/feat/targets feat/targets
```

Confirmar ignore antes de criar:

```bash
git check-ignore -q .worktrees
```

## Regras

- Um worktree por tarefa. Não reutilizar o diretório pra outra feat sem `git worktree remove`.
- `.worktrees/` está no `.gitignore`. Não commitar o conteúdo.
- `./.jdt/` é por worktree; não compartilhar com o clone principal.
- Features independentes (sem import cruzado): worktree, não stash-hopping e não a mesma stack.

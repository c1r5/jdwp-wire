# Diagrama

O `jdt attach` corre esta sequência sozinho, do pull ao forward, e pula a etapa que já está feita. `devices`, `pull`, `patch`, `install` e `reset` são as mesmas etapas, soltas.

`com.alvo` é o package. Com mais de um aparelho, `-s <serial>` no pull, no install e no attach.

```text
┌─ 1. devices ──────────────────────────────────────────┐
│ jdt devices                                           │
└────────────────────────────┬──────────────────────────┘
                             │
                             v
┌─ 2. pull ─────────────────────────────────────────────┐
│ jdt pull --decode com.alvo                            │
│ → .jdt/com.alvo/apk/base.apk                          │
│ → .jdt/com.alvo/decode                                │
└────────────────────────────┬──────────────────────────┘
                             │
                             v
┌─ 3. patch ────────────────────────────────────────────┐
│ jdt patch .jdt/com.alvo/decode                        │
│ debuggable=true + network security config (user CA)   │
└────────────────────────────┬──────────────────────────┘
                             │
                             v
┌─ 4. install ──────────────────────────────────────────┐
│ jdt install .jdt/com.alvo/decode                      │
│ apktool rebuild, debug sign, adb install -r -d        │
└────────────────────────────┬──────────────────────────┘
                             │
                             v
┌─ 5. attach ───────────────────────────────────────────┐
│ jdt attach com.alvo --port 8700 --studio              │
│ pull, decode, patch, sign, install, forward           │
│ → .jdt/com.alvo/idea                                  │
│ → 15:04:05 [ok] [attach] 127.0.0.1:8700               │
└────────────────────────────┬──────────────────────────┘
                             │
                             v
┌─ 6. breakpoint ───────────────────────────────────────┐
│ Android Studio → Remote JVM Debug                     │
│ host 127.0.0.1, port 8700                             │
└────────────────────────────┬──────────────────────────┘
                             │
                             v
┌─ 7. reset ────────────────────────────────────────────┐
│ jdt reset com.alvo                                    │
│ clear-debug-app and remove the JDWP forward           │
└───────────────────────────────────────────────────────┘
```

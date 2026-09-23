# jdwp-wire

CLI (`jdt`) that turns an Android APK/package into a JDWP session usable from Android Studio / jdb.

This is not MobSF. It is not Frida-first. The main path is **smali/Java + JVM debugger**.

Status: **MVP in progress**. Wired so far: `jdt devices`, `jdt apps`, `jdt pull`, `jdt patch`, `jdt install`, `jdt attach`, `jdt reset`, `jdt targets`. `jdt attach --studio` writes `.jdt/<pkg>/idea` after the JDWP forward and does not launch Android Studio. `--crypto` on targets is v1. `jdt apps` lists installed packages (PID when running); `jdt pull <index>` uses that list.

Happy path (`com.alvo` is the package). With more than one device, pass `-s <serial>` on `pull`, `install`, and `attach`.

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
│ set-debug-app -w, launch, adb forward, probe          │
│ → .jdt/com.alvo/idea                                  │
│ → attach: localhost:8700                              │
└────────────────────────────┬──────────────────────────┘
                             │
                             v
┌─ 6. breakpoint ───────────────────────────────────────┐
│ Android Studio → Remote JVM Debug                     │
│ host localhost, port 8700                             │
└────────────────────────────┬──────────────────────────┘
                             │
                             v
┌─ 7. reset ────────────────────────────────────────────┐
│ jdt reset com.alvo                                    │
│ clear-debug-app and remove the JDWP forward           │
└───────────────────────────────────────────────────────┘
```

```
go build -o jdt .
go install .          # binário GOBIN: jdwp-wire (module path)
./jdt --help
./jdt devices
./jdt devices --json
./jdt apps
./jdt apps --json
./jdt pull com.alvo
./jdt pull 2
./jdt pull --decode com.alvo
./jdt pull ./app.apk --package com.alvo
./jdt patch .jdt/com.alvo/decode
./jdt patch --apk ./app.apk --package com.alvo
./jdt install com.alvo
./jdt install .jdt/com.alvo/decode
./jdt install .jdt/com.alvo/apk/base.apk
./jdt attach com.alvo --port 8700
./jdt attach com.alvo --port 8700 --studio
./jdt reset com.alvo
./jdt targets com.alvo
./jdt targets .jdt/com.alvo/decode --http
./jdt targets com.alvo --json
```

Product contract: [`AGENTS.md`](AGENTS.md). Repo workflow: [`.grok/rules/development-rules.md`](.grok/rules/development-rules.md).

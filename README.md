# jdwp-wire

CLI (`jdt`) that turns an Android APK/package into a JDWP session usable from Android Studio / jdb.

This is not MobSF. It is not Frida-first. The main path is **smali/Java + JVM debugger**.

Status: **MVP wired**. `jdt devices`, `jdt apps`, `jdt pull`, `jdt patch`, `jdt install`, `jdt attach`, `jdt reset`, `jdt targets`. `jdt attach` always pulls and decodes into `.jdt/<pkg>`, patches whatever is still missing, re-signs, and reinstalls, then forwards JDWP. An apktool tree or a patch that is already applied is skipped on its own; the later steps still run. When the official Android CLI is on PATH, install and launch use `android run --debug`; otherwise `adb` + `am`. `jdt attach --studio` writes `.jdt/<pkg>/idea` after the forward and does not launch Android Studio. `--bypass` and `--script` load a Frida companion after the forward and follow its filtered log; `-d` only writes `.jdt/<pkg>/frida/YYYY-MM-DD.log`. SIGINT/SIGTERM, once launch has started, force-stops the package, clears the debug app, and removes the forward; the Frida CLI dies with the command and `frida-server` stays up. An attach that returns without a signal leaves the app waiting for the debugger. `--crypto` on targets is v1. `jdt apps` lists third-party packages (PID when running, running apps first); `--system` includes system packages. `jdt pull <index>` uses that list, and `jdt pull --system <index>` uses the `--system` list.

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
│ repack if not debuggable, then forward + probe        │
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

```
go build -o jdt .
go install .          # binário GOBIN: jdwp-wire (module path)
./jdt --help
./jdt devices
./jdt devices --json
./jdt apps
./jdt apps --system
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
./jdt attach com.alvo --bypass
./jdt attach com.alvo --script sslpinning-bypass,./custom.js -d
./jdt reset com.alvo
./jdt targets com.alvo
./jdt targets .jdt/com.alvo/decode --http
./jdt targets com.alvo --json
```

Product contract: [`AGENTS.md`](AGENTS.md).

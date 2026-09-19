# jdwp-wire

CLI (`jdt`) that turns an Android APK/package into a JDWP session usable from Android Studio / jdb.

This is not MobSF. It is not Frida-first. The main path is **smali/Java + JVM debugger**.

Status: **MVP in progress**. Wired so far: `jdt devices`, `jdt pull`, `jdt patch`, `jdt install`. `jdt attach` is not wired yet.

```
go build -o jdt ./cmd/jdt
./jdt --help
./jdt devices
./jdt devices --json
./jdt pull com.alvo
./jdt pull --decode com.alvo
./jdt pull ./app.apk --package com.alvo
./jdt patch .jdt/com.alvo/decode
./jdt patch --apk ./app.apk --package com.alvo
./jdt install com.alvo
./jdt install .jdt/com.alvo/decode
./jdt install .jdt/com.alvo/apk/base.apk
```

Product contract: [`AGENTS.md`](AGENTS.md). Repo workflow: [`.grok/rules/development-rules.md`](.grok/rules/development-rules.md).

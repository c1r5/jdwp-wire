# jdwp-wire

CLI (`jdt`) that turns an Android APK/package into a JDWP session usable from Android Studio / jdb.

This is not MobSF. It is not Frida-first. The main path is **smali/Java + JVM debugger**.

Status: **MVP in progress**. `jdt devices` lists adb devices (table or `--json`). Other pipeline commands are not wired yet.

```
go build -o jdt ./cmd/jdt
./jdt --help
./jdt devices
./jdt devices --json
```

Product contract: [`AGENTS.md`](AGENTS.md). Repo workflow: [`.grok/rules/development-rules.md`](.grok/rules/development-rules.md).

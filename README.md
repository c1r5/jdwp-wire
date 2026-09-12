# jdwp-wire

CLI (`jdt`) that turns an Android APK/package into a JDWP session usable from Android Studio / jdb.

This is not MobSF. It is not Frida-first. The main path is **smali/Java + JVM debugger**.

Status: **scaffold**. The binary builds and prints help. Pipeline commands are not wired yet.

```
go build -o jdt ./cmd/jdt
./jdt --help
```

Product contract: [`CONTEXT.md`](CONTEXT.md). Repo workflow: [`development-rules.md`](development-rules.md).

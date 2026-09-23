# jdwp-wire

CLI para transformar um APK/package Android numa sessão JDWP usável no Android Studio / jdb, e depois extrair request viva do breakpoint.

Não é MobSF. Não é Frida-first. O caminho principal é **smali/Java + debugger JVM**.

Origem: ritual manual (apktool → Android Studio → breakpoint → reconstruir request) + protótipo `c1r5/android_debug_tool` (2023), que só fazia `set-debug-app` + `adb forward tcp:PORT jdwp:PID`.

---

## Tese

Frida loga string. JDWP deixa inspecionar o objeto já montado (`okhttp3.Request`, buffer crypto, header map).

O produto não é o forward. É o pipeline até o attach + (no v1) o dump estruturado do frame.

Uso legítimo: app próprio, pentest autorizado, research / bug bounty no escopo.

---

## Estado herdado (`android_debug_tool`)

O que existe hoje:

- `adt static -t pkg -p 8000` → `am set-debug-app -w` → launch/Frida spawn → `adb forward` → espera Enter
- listar packages via Frida
- wrapper ADB com estado estático global
- Frida loader incompleto

O que **não** está no repo: o `.sh` que automatizava o resto do procedimento.

Limitações conhecidas: um device só, app não-debuggable, split APK, PID de processo errado (`:id` / isolated), sem decode, sem projeto Studio, sem captura de request.

Reaproveitar a ideia. Não reaproveitar o desenho de `Adb`/`AdbShell` estático.

---

## Pipeline (visão)

```
apk | package
  → device check
  → pull (split-aware)
  → decode (apktool)
  → patch (debuggable + NSC user CA)
  → rebuild + sign + install
  → set-debug-app -w
  → launch + wait JDWP
  → adb forward tcp:PORT jdwp:PID
  → attach hint (Studio / jdb)
  → [v1] watch frame → HAR / JSON
```

Cada etapa é módulo isolado. O CLI só orquestra.

Onde a [Android CLI](https://developer.android.com/tools/agents/android-cli) oficial cobrir o passo (install/debug/split/device), preferir shell-out a reimplementar `adb install` / `am start`. Ver seção abaixo.

---

## CLI (formato alvo)

Binário: `jdt`

```
jdt devices
jdt apps
jdt pull     <index|pkg|apk> [--decode]
jdt patch    <decoded_dir|--apk>
jdt install  <apk|decoded_dir|pkg>
jdt attach   <pkg> [--port 8700] [--studio] [--no-patch]
jdt targets  <pkg|decoded> [--http] [--crypto]
jdt watch    --port 8700 [--dump-okhttp]   # v1
jdt reset    <pkg>
```

Atalho do dia a dia (MVP):

```
jdt attach com.alvo --port 8700 --studio
```

Faz o pipeline quando o app instalado não é debuggable e imprime o skip quando já é (`[skip] patch: already debuggable`). `android run --debug` só entra depois desse repack, quando a CLI 1.0 está no PATH. App já debuggable segue em `adb` + `am`, porque não há APK recém-assinado para `--apks`.

---

## Módulos

| Módulo    | Papel                                      | MVP | v1 |
|-----------|--------------------------------------------|-----|----|
| `device`  | serial, root/userdebug, `pidof` estável, packages instalados | sim | multi-device explícito |
| `apps`    | `jdt apps`: IDX/PID/nome/package; índice do `pull` | sim | label de `@string` |
| `apk`     | pull, splits, apktool, sign                | pull + decode + sign + install-multiple (sem merge) | AAB |
| `patch`   | `debuggable=true`, NSC user CA             | sim | extractNativeLibs, keep signature se possível |
| `jdwp`    | set-debug-app, wait, forward, healthcheck  | sim | retry / process spawn vs attach |
| `project` | esqueleto IntelliJ em `.jdt/<pkg>/idea` (content root = decode) + Remote JVM Debug `localhost:PORT` | sim (`attach --studio` escreve depois do forward) | JADX sources opcional |
| `targets` | sinks HTTP/crypto no smali/java            | lista OkHttp/Retrofit/HttpURLConnection | Cipher / pinning classes |
| `capture` | dump JDWP do frame → JSON/HAR              | não | `watch --dump-okhttp` |
| `frida`   | companion unpin / hide-debugger            | não | `--unpin` opcional |
| `android` | shell-out p/ Android CLI se estiver no PATH | detect + `run --debug` | emulator, layout |

Frida não é o caminho principal. Entra só quando JDWP sozinho não segura (pinning, anti-debug).

### `project` vs o resto

`workspace` reserva paths (`idea/`, `decode/`, …) e não escreve conteúdo.
`apk` faz pull/decode/sign/install. `jdwp` faz set-debug-app, launch, forward, probe.
`project` **só** materializa o diretório que o Android Studio abre.

MVP: `.jdt/<pkg>/idea` com `decode.iml` + `.idea/{misc,modules}.xml` + `.idea/runConfigurations/Remote_Debug.xml`. O smali **não** se copia — o iml aponta `../decode`. Sem Gradle, sem plugin Android, sem escolher o JDK da máquina, sem instalar smalidea.

Não há `jdt project`. `jdt attach --studio` chama `project.Write` depois do forward e imprime o path de `idea/`. Sem a flag, imprime skip. Sem decode (`AndroidManifest.xml`), imprime skip e o attach segue. `project` não lança o IDE.

Patch destrutivo e Integrity continuam a ser problema de `patch`/`apk`, não deste módulo.

---

## Android CLI (oficial, 2026)

Ferramenta nova do Android Studio / Google pra fluxo agent-first no terminal (`android` no PATH). Stable 1.0. Não substitui o jdwp-wire — cobre a camada *device/deploy* que a gente ia reinventar.

Encaixe útil:

| Comando oficial | O que evita no `jdt` |
|-----------------|----------------------|
| `android run --debug --apks=a.apk` | install + wait-for-debugger no lugar de `am set-debug-app -w` + start manual |
| `android run --apks=base.apk,split_config.xxhdpi.apk,...` | split sem merge caseiro |
| `android run --device=<serial>` | multi-device |
| `android run --activity=...` / `--type=SERVICE` | launch do componente certo |
| `android emulator list/start/stop` | não criar AVD manager |
| `android layout` / `screen capture` / `screen resolve` | [v1+] dirigir UI até o sink, sem Appium |
| `android init` + skills | se um agente for orquestrar o `jdt` |

MVP: se `android` for a CLI 1.0 (`run --apks` / `--debug`), `jdt attach` usa `android run --debug --apks=... --install-options=-r,-d` depois do patch. O binário antigo do SDK conta como ausente. Sem a CLI, fallback `adb install` + `am set-debug-app -w`. App já debuggable não patcheia e o launch fica no fallback. Probe da porta nos dois caminhos. Não tornar Android CLI dependência dura.

Não usar no MVP: `android create`, `docs`, Journeys, skills. Isso é dev de app greenfield, não RE.

Docs: https://developer.android.com/tools/agents/android-cli

---

---

## MVP (2 semanas, shippable)

Objetivo: **um comando deixa o debugger attachável e o smali aberto no Studio**.

Inclui:

1. Device único via `adb` (falha clara se 0 ou N devices sem `-s`).
2. Reescrita do attach sem estado global no ADB.
3. `pull` do package instalado (base.apk no mínimo).
4. Patch `android:debuggable="true"` no manifest + rebuild + sign debug + install (`-r` / `-d` se necessário).
5. Launch debugável: depois do repack, `android run --debug --apks=...` se a CLI oficial 1.0 estiver instalada; senão `am set-debug-app -w` + start + `forward tcp:PORT jdwp:PID`. App já debuggable não repacka e usa o fallback. Probe da porta nos dois caminhos.
6. `jdt reset` (clear-debug-app + remove forward).
7. Módulo `project`: escreve o esqueleto IntelliJ em `.jdt/<pkg>/idea` (Remote Debug `localhost:PORT`, content root = decode já existente). `jdt attach --studio` chama isto depois do forward. Sem decode, imprime skip e o attach segue.
8. `jdt targets --http`: grep/parse raso de OkHttp / Retrofit / `HttpURLConnection` no decode; imprime `classe#metodo`.
9. Log em texto: cada passo, skip, e o one-liner de attach.
10. `jdt apps`: packages instalados (PID se o processo existe). `jdt pull` aceita o IDX dessa lista, além de package ou APK local. O nome é o label não-localizado; label de resource fica pro package.

Não inclui no MVP:

- dump automático de Request
- Frida
- Flutter / RN / Unity
- watcher de Play Store
- GUI
- multi-device paralelo
- unpacker de packer

Critério de pronto: num app debugável (ou patcheado) de teste, `jdt attach --studio` + Attach to process no Android Studio para no breakpoint de um método listado por `targets`.

---

## v1

Objetivo: **do breakpoint sai request estruturada**, e o attach sobrevive aos casos chatos.

Inclui:

1. `jdt watch --port --dump-okhttp`  
   Conecta no JDWP (jdb/JDI), para em métodos conhecidos (`Request.Builder.build`, `RealCall.execute` / `enqueue`, equivalentes Retrofit).  
   Lê url / method / headers / body e grava JSON + HAR.

2. `targets --crypto` (Cipher, SecretKeySpec, Mac) — lista, ainda sem dump genérico.

3. AAB. Split APK já é pull de todos os paths + `adb install-multiple` (sem merge).

4. PID certo: processo default vs `:remote` / isolated; flag `--process`.

5. `--unpin`: script Frida companion só pra pinning comum; JDWP continua no centro.

6. Saúde do attach: timeout, “waiting for debugger”, re-forward se o processo morrer.

7. `--no-patch` para app que já é debuggable / build userdebug.

8. Export `targets` como script jdb (`stop in ...`).
9. Opcional: `android layout` / `screen resolve` só como helper pra chegar no fluxo que dispara o request (não é o produto).

Ainda fora do v1 (backlog consciente):

- OpenAPI gerado do dump
- Flutter Dio / Cronet / custom TLS nativo
- anti-debug avançado / packer unpack
- grafo classe→sink
- MCP server

---

## Decisões

- Go 1.23+ CLI (`jdt`). Wrappers de `adb`, `apktool`, `apksigner`/`uber-apk-signer`, `jdb`. Sem daemon. Sem Python no core.
- Android CLI (`android`) é opcional e detectada em runtime. Nunca assumir que o binário `android` é o SDK antigo (`android`/`sdkmanager` pre-studio).
- Saída humana no TTY + `--json` nas queries (`apps`, `targets`, `watch`).
- Workspace local `./.jdt/<package>/` (apk, decode, patched, idea, captures).
- Patch é destrutivo no APK instalado (vira build debug assinado por nós). Deixar isso explícito no help.
- Não mentir compatibilidade: release Play com Integrity / anti-tamper pode quebrar no patch. MVP documenta o fail, não contorna.

---

## Fluxo mental do usuário

1. Plug device, USB debug on.
2. `jdt attach com.alvo --studio`
3. Abre `.jdt/<pkg>/idea` no Android Studio (não a pasta `decode/`).
4. Attach Remote Debugger na porta impressa.
5. Break no método que `targets` apontou (ou no que já conhece).
6. Usa o app. No hit, inspeciona o objeto Request.
7. [v1] ou deixa o `watch` gravar o HAR sozinho.

Se o attach falhar: `jdt reset` e ler o passo que skipou (não debuggable, PID errado, porta ocupada).

---

## Nomes

- Repo / binário de trabalho: `jdwp-wire` / `jdt`
- Repo antigo: referência histórica, não contrato de CLI (`adt static` some)

Quando este arquivo divergir do código, o código mente — atualizar o AGENTS.md no mesmo PR.

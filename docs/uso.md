# Uso

`jdt --help` lista os comandos. No dia a dia eu uso o attach, porque ele já corre o pipeline: pull, decode, patch, install, launch e forward.

```bash
jdt attach br.com.example --port 8700 --studio
```

Com mais de um aparelho, `-s <serial>` no attach, no pull e no install. Um aparelho só dispensa o serial.

Cada etapa imprime uma linha quando termina, antes do próximo comando bloqueante:

```text
15:04:01 [ok] [pull] base.apk
15:04:12 [skip] [patch] already debuggable
15:04:40 [ok] [attach] 127.0.0.1:8700
```

`--json` troca esse texto por JSON nos comandos que aceitam a flag.

## Attach

```text
jdt attach <pkg>
```

O comando começa no pull e no decode e termina no install. O `adb forward` só roda se o serial ainda estiver ligado depois disso. Se o cabo sair no meio, o forward não roda.

Árvore apktool que já existe, com manifest e `apktool.yml`, não baixa o APK de novo. Manifest já com `android:debuggable="true"`, ou network security config que já confia na CA de usuário, pula só esse pedaço do patch. O install roda mesmo assim, e o app no aparelho passa a ser o build de debug assinado pelo `jdt`. A keystore fica em `.jdt/debug.keystore`. Checagem de assinatura no próprio app pode recusar esse build.

Se a assinatura não bater com a que está instalada (`INSTALL_FAILED_UPDATE_INCOMPATIBLE`), o `jdt` faz `adb uninstall` e tenta de novo. Os dados do app somem. Se a sessão logada importa, eu anoto antes.

Com a Android CLI oficial 1.0 no PATH, install e launch saem de `android run --debug --apks=...`. O `android` velho do SDK conta como ausente. Sem a CLI 1.0, o caminho é `adb install` e `am set-debug-app -w`.

A porta padrão é 8700 (`--port`). O forward escuta em IPv4:

```text
adb -s <serial> forward tcp:8700 jdwp:<pid>
```

O PID é o do processo cujo nome é o package. O debugger do host liga em `127.0.0.1:8700`. A confirmação é `adb forward --list`. A primeira conexão TCP nesse socket é o handshake JDWP, então o attach não abre a porta. Deixa pro Android Studio.

### Studio

`--studio` escreve `.jdt/<pkg>/idea` depois do forward. Eu abro essa pasta no Android Studio, na mão.

```text
.jdt/br.com.example/idea/
├── decode.iml
└── .idea/
    ├── misc.xml
    ├── modules.xml
    └── runConfigurations/Remote_Debug.xml
```

O content root do módulo aponta pra `../decode`. O smali fica lá. A run configuration `Remote_Debug` usa `127.0.0.1` e a porta que o attach imprimiu. O attach na IDE é o Debug dessa configuration.

Sem `AndroidManifest.xml` no decode, o studio imprime skip e o forward continua valendo. Sem a flag, o log é um skip pedindo `--studio`. Dá pra rodar o attach de novo com a flag. O que já está feito não baixa o APK outra vez.

### Frida no attach

`--bypass` carrega `antiroot-bypass`, `antidebug-bypass` e `sslpinning-bypass` depois do forward. `--script` aceita esses nomes ou um caminho `.js`, separado por vírgula, e pode repetir. O app continua esperando o debugger.

O `jdt` sobe o `frida-server` com `su` quando ele está parado em `/data/local/tmp/frida-server`.

A saída filtrada vai pra `.jdt/<pkg>/frida/AAAA-MM-DD.log`. Sem `-d`, esse fluxo segue no stderr depois da linha do attach. `-d` (`--detach`) grava o arquivo e o comando retorna.

O `load.js` avalia o script quando o class loader da `Application` existe. Se `currentApplication()` já devolve a instância, roda na hora. Enquanto o processo espera o debugger, o script entra em `Application.attach`, antes de `onCreate`. O que esses scripts fazem está em [analise.md](analise.md).

`SIGINT` e `SIGTERM` cancelam o comando. Se o launch já começou, o `jdt` dá `am force-stop`, limpa o `set-debug-app` e remove o forward antes de sair. O CLI do Frida morre junto. No `-d`, o `frida-session` faz esse fechamento. O `frida-server` continua. Sem sinal, o attach que retorna deixa o app esperando o debugger.

Flags:

```text
-s, --serial    serial do adb (obrigatório com mais de um aparelho)
    --port      porta TCP local do forward (padrão 8700)
    --studio    escreve .jdt/<pkg>/idea (não abre o Android Studio)
    --bypass    antiroot-bypass, antidebug-bypass e sslpinning-bypass
    --script    nome embutido ou caminho .js, separado por vírgula, repetível
-d, --detach    grava o log do Frida e não acompanha no stderr
    --json      saída JSON
```

## Os outros comandos

Uma etapa só, quando eu quero parar antes do attach.

`jdt devices` lista o aparelho ligado. Sem device, ou com vários e sem `-s`, ele para.

`jdt apps` lista pacote de terceiro (`pm list packages -3`), com PID quando o processo existe. Quem está em execução vem primeiro. Dentro do grupo, ordena por nome e depois por package. `--system` inclui pacote de sistema. O IDX começa em 1 e é o número que o `pull` aceita. A coluna NAME é o label não localizado, quando o `dumpsys package` imprime um. Senão, NAME é o package. `--json` devolve a mesma lista.

`jdt pull <index|pkg|apk>` baixa pelo IDX dessa lista, pelo package, ou copia um APK que já está no host (`--package` obrigatório no arquivo). Por baixo é o `pm path`, sem o hash da pasta de instalação. Split (`split_config.*`) cai junto em `.jdt/<pkg>/apk/`. `--decode` roda `apktool d` no base e escreve `.jdt/<pkg>/decode`. `--system` resolve o IDX contra `jdt apps --system`.

`jdt patch [decoded_dir]` mexe no decode e para. O manifest ganha `android:debuggable="true"`. O network security config passa a confiar na CA de usuário (`<certificates src="user" />`). No XML que o `jdt` cria, `cleartextTrafficPermitted` fica `true`. Se o manifest já apontava pra um NSC, o patch acrescenta o certificado de usuário nesse arquivo. `--apk` com `--package` decodifica e aplica o patch a partir do APK. O patch não instala.

`jdt install <apk|decoded_dir|pkg>` rebuilda com apktool quando a entrada é a pasta do decode, assina com a keystore de debug e instala com `adb install -r -d`. `-r` substitui. `-d` permite downgrade de versionCode. Base mais splits vão de `adb install-multiple -r -d`, cada um assinado. Também aceita o package, usando o que já está em `.jdt/`, ou um APK. `--package` entra quando o decode está fora de `.jdt/<pkg>/decode`.

`jdt reset <pkg>` roda `am clear-debug-app` e `adb forward --remove` na porta da sessão (`--port`, padrão 8700). O app patcheado continua instalado. O que solta é o wait-for-debugger.

`jdt targets <pkg|decoded>` lista chamada HTTP no decode (OkHttp, Retrofit, `HttpURLConnection`) como `classe#metodo`. Aceita o package ou a pasta do decode. `--http` liga esse modo e já vem ligado. `--json` devolve `dir` e a lista.

O texto de cada comando também sai em `jdt <comando> --help`.

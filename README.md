# jdwp-wire

Uma coisa que eu gosto de fazer na busca por falhas em aplicativo Android é análise dinâmica com JDWP e o Android Studio. Dali eu mapeio a chamada com o objeto já montado na mão e escrevo o script de Frida com bem menos chute. Chegar nesse ponto era um ritual de apktool, patch, assinatura, install e `adb forward`. O `jdt` faz esse ritual.

O caminho é smali, Java e o debugger da JVM. Frida entra depois do forward, quando o breakpoint já mostrou o que interceptar. O relato mais longo está em [notes.c1r5.dev/posts/jdwp-wire](https://notes.c1r5.dev/posts/jdwp-wire/).

```bash
go build -o jdt .
go install .          # binário no GOBIN: jdwp-wire (module path)
jdt attach br.com.example --port 8700 --studio
```

O attach puxa o APK pra `.jdt/<package>/`, decodifica, põe `android:debuggable="true"` e a CA de usuário no network security config quando o decode ainda não tem isso, reassina e reinstala. Etapa já feita é skip só dela. O install corre na mesma. Se a Android CLI 1.0 estiver no PATH, o launch sai de `android run --debug`. Sem ela, `adb install` e `am set-debug-app -w`. No fim o forward escuta em IPv4:

```text
15:04:40 [ok] [attach] 127.0.0.1:8700
```

Eu abro `.jdt/br.com.example/idea` no Android Studio. A run configuration `Remote_Debug` aponta pra `127.0.0.1` na porta impressa. A primeira conexão TCP nessa porta é o handshake JDWP, então o attach deixa o socket quieto até o Studio ligar.

O desenho dos comandos está em [docs/diagram.md](docs/diagram.md). O uso, com o attach no centro, está em [docs/uso.md](docs/uso.md). Análise dinâmica e o hook de Frida estão em [docs/analise.md](docs/analise.md).

Workspace local: `./.jdt/<package>/`. Nada disso vai pro git.

Contrato do produto: [AGENTS.md](AGENTS.md). Fluxo do repo: [.grok/rules/development-rules.md](.grok/rules/development-rules.md).

# Análise dinâmica

No JDWP eu paro num método e olho o objeto que o app já montou. URL, header e body estão nesse objeto, e dali eu escrevo o script de Frida com bem menos chute.

O `jdt targets` lê o smali em `.jdt/<pkg>/decode` e imprime `classe#metodo` pra OkHttp, Retrofit e `HttpURLConnection`. Eu abro `.jdt/<pkg>/idea` no Android Studio, ligo a `Remote_Debug` em `127.0.0.1` na porta do attach e quebro nesse método. "Attach Debugger to Android Process" e o Debug de um projeto Android normal falam com outra sessão. O uso do attach está em [uso.md](uso.md).

O processo sobe em wait-for-debugger. Enquanto eu não ligo o Studio, o app fica parado.

## Hook com Frida

Frida entra quando o breakpoint já mostrou o que interceptar, ou quando o app recusa o depurador, o root ou o proxy. No `jdt` ele é companheiro do attach, depois do forward.

`--bypass` carrega três scripts embutidos, nesta ordem:

- `antiroot-bypass` esconde checagem comum de root e Magisk.
- `antidebug-bypass` esconde debugger conectado e `TracerPid`.
- `sslpinning-bypass` solta pinning TLS comum em Java, pro app falar com a CA de usuário que o patch colocou no network security config.

`--script` carrega um desses nomes ou um `.js` meu, no mesmo ponto.

O `jdt` escreve um `load.js` e o Frida carrega só esse arquivo. O `load.js` avalia o script quando o class loader da `Application` existe. Se `ActivityThread.currentApplication()` já devolve a instância, o script roda na hora, com o class loader dela. Enquanto o processo espera o debugger, `handleBindApplication` já está na stack e `currentApplication()` ainda vem nulo. O script entra em `Application.attach`, com o class loader do contexto, e o `attach` original segue. Isso é antes de `onCreate`. O `Java.perform` do script roda nesse momento, depois que o Studio continua.

A linha que o script manda pro `console.log` (por exemplo `sslpinning-bypass armed`) cai filtrada em `.jdt/<pkg>/frida/AAAA-MM-DD.log`. Sem `-d`, o attach acompanha esse fluxo no stderr. Com `-d`, grava o arquivo e volta. `SIGINT` ou `SIGTERM` mata o CLI do Frida (ou o `frida-session`, no `-d`) e fecha o app. O `frida-server` em `/data/local/tmp/frida-server` continua. Se ele estava parado, o attach sobe ele com `su` antes de carregar o script.

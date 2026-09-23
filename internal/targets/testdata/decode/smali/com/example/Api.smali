.class public Lcom/example/Api;
.super Ljava/lang/Object;

.method public load()V
    .locals 2

    invoke-direct {v0}, Lokhttp3/Request$Builder;-><init>()V

    invoke-virtual {v0}, Lokhttp3/Request$Builder;->build()Lokhttp3/Request;

    move-result-object v0

    invoke-virtual {v1, v0}, Lokhttp3/OkHttpClient;->newCall(Lokhttp3/Request;)Lokhttp3/Call;

    move-result-object v0

    invoke-interface {v0}, Lokhttp3/Call;->execute()Lokhttp3/Response;

    return-void
.end method

.method public enqueue(Lretrofit2/Call;)V
    .locals 1

    invoke-interface {p1, p2}, Lretrofit2/Call;->enqueue(Lretrofit2/Callback;)V

    return-void
.end method

.method public constructor <init>()V
    .locals 0

    invoke-direct {p0}, Ljava/lang/Object;-><init>()V

    return-void
.end method

.method public await(Lretrofit2/Call;)Ljava/lang/Object;
    .locals 1

    invoke-static {p1, p2}, Lretrofit2/KotlinExtensions;->await(Lretrofit2/Call;Lkotlin/coroutines/Continuation;)Ljava/lang/Object;

    return-object v0
.end method

.method public noop()V
    .locals 0

    # invoke-virtual {v0}, Lokhttp3/Request$Builder;->build()Lokhttp3/Request;

    const-string v0, "Lokhttp3/Request$Builder;->build("

    return-void
.end method

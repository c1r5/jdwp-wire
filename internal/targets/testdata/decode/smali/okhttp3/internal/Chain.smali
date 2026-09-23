.class public Lokhttp3/internal/Chain;
.super Ljava/lang/Object;

.method public proceed()Lokhttp3/Response;
    .locals 1

    invoke-virtual {v0}, Lokhttp3/Request$Builder;->build()Lokhttp3/Request;

    return-object v0
.end method

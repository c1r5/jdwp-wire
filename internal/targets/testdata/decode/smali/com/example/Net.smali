.class public Lcom/example/Net;
.super Ljava/lang/Object;

.method public open()V
    .locals 1

    invoke-virtual {v0}, Ljava/net/URL;->openConnection()Ljava/net/URLConnection;

    move-result-object v0

    invoke-virtual {v0}, Ljava/net/HttpURLConnection;->getResponseCode()I

    return-void
.end method

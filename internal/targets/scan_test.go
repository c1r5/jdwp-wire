package targets

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestScanHTTP(t *testing.T) {
	t.Parallel()
	root := filepath.Join("testdata", "decode")
	res, err := ScanHTTP(root)
	if err != nil {
		t.Fatal(err)
	}
	if res.Dir != root {
		t.Fatalf("Dir=%s", res.Dir)
	}
	got := map[string][]Kind{}
	for _, h := range res.Hits {
		got[h.Class+"#"+h.Method] = h.Kinds
	}
	want := map[string][]Kind{
		"com.example.Api#load":      {KindOKHTTP},
		"com.example.Api#enqueue":   {KindRetrofit},
		"com.example.Api#await":     {KindRetrofit},
		"com.example.Net#open":      {KindURLConnection},
		"com.example.JavaApi#fetch": {KindOKHTTP},
	}
	if len(got) != len(want) {
		t.Fatalf("hits=%v", got)
	}
	for k, kinds := range want {
		if len(got[k]) != len(kinds) || got[k][0] != kinds[0] {
			t.Fatalf("%s kinds=%v", k, got[k])
		}
	}
	prev := ""
	for _, h := range res.Hits {
		key := h.Class + "#" + h.Method
		if key < prev {
			t.Fatalf("unsorted %s after %s", key, prev)
		}
		prev = key
	}
}

func TestScanHTTPEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	res, err := ScanHTTP(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hits) != 0 {
		t.Fatalf("hits=%v", res.Hits)
	}
}

func TestScanHTTPMissing(t *testing.T) {
	t.Parallel()
	_, err := ScanHTTP(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestScanHTTPFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "f.smali")
	if err := os.WriteFile(path, []byte(".class public Lcom/example/A;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ScanHTTP(path)
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("err=%v", err)
	}
}

func TestDecodeDirPackage(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	decode := filepath.Join(cwd, ".jdt", "com.alvo", "decode")
	if err := os.MkdirAll(decode, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := DecodeDir(cwd, "com.alvo")
	if err != nil {
		t.Fatal(err)
	}
	if got != decode {
		t.Fatalf("got %s", got)
	}
}

func TestDecodeDirExplicit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	got, err := DecodeDir(t.TempDir(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Fatalf("got %s", got)
	}
}

func TestDecodeDirMissingPackage(t *testing.T) {
	t.Parallel()
	_, err := DecodeDir(t.TempDir(), "com.missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, ErrUsage) {
		t.Fatalf("missing decode is not usage: %v", err)
	}
}

func TestScanHTTPMergesKinds(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	body := "" +
		".class public Lcom/example/Both;\n" +
		".super Ljava/lang/Object;\n" +
		".method public go()V\n" +
		"    .locals 1\n" +
		"    invoke-virtual {v0}, Lokhttp3/Request$Builder;->build()Lokhttp3/Request;\n" +
		"    invoke-virtual {v0}, Ljava/net/URL;->openConnection()Ljava/net/URLConnection;\n" +
		"    invoke-virtual {v0}, Lokhttp3/Request$Builder;->build()Lokhttp3/Request;\n" +
		"    return-void\n" +
		".end method\n"
	if err := os.WriteFile(filepath.Join(dir, "Both.smali"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := ScanHTTP(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hits) != 1 {
		t.Fatalf("hits=%v", res.Hits)
	}
	h := res.Hits[0]
	if h.Class != "com.example.Both" || h.Method != "go" {
		t.Fatalf("%+v", h)
	}
	if len(h.Kinds) != 2 || h.Kinds[0] != KindURLConnection || h.Kinds[1] != KindOKHTTP {
		t.Fatalf("kinds=%v", h.Kinds)
	}
}

func TestDecodeDirRejectsFileAndPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	file := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeDir(dir, file); !errors.Is(err, ErrUsage) {
		t.Fatalf("file err=%v", err)
	}
	if _, err := DecodeDir(dir, filepath.Join(dir, "nope")); err == nil || errors.Is(err, ErrUsage) {
		t.Fatalf("missing path err=%v", err)
	}
	if _, err := DecodeDir(dir, "com/alvo"); !errors.Is(err, ErrUsage) {
		t.Fatalf("slash package err=%v", err)
	}
}

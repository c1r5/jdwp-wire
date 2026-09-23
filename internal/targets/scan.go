// Package targets lists HTTP call sites in an apktool decode tree.
package targets

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/c1r5/jdwp-wire/internal/workspace"
)

// ErrUsage is a bad argument (not a missing decode tree).
var ErrUsage = errors.New("usage")

// Kind is the HTTP stack a call site touches.
type Kind string

const (
	KindOKHTTP        Kind = "okhttp"
	KindRetrofit      Kind = "retrofit"
	KindURLConnection Kind = "httpurlconnection"
)

// Hit is one breakpoint: the method that invokes a sink.
type Hit struct {
	Class  string
	Method string
	Kinds  []Kind
}

// Result is a scan of one decode directory.
type Result struct {
	Dir  string
	Hits []Hit
}

var (
	okhttpSinks = []string{
		"Lokhttp3/Request$Builder;->build(",
		"Lokhttp3/OkHttpClient;->newCall(",
		"Lokhttp3/Call;->execute(",
		"Lokhttp3/Call;->enqueue(",
		"Lokhttp3/RealCall;->execute(",
		"Lokhttp3/RealCall;->enqueue(",
	}
	retrofitSinks = []string{
		"Lretrofit2/Call;->execute(",
		"Lretrofit2/Call;->enqueue(",
		"Lretrofit2/KotlinExtensions;->await(",
		"Lretrofit2/KotlinExtensions;->awaitResponse(",
	}
	urlSinks = []string{
		"Ljava/net/HttpURLConnection;->",
		"Ljavax/net/ssl/HttpsURLConnection;->",
		"Ljava/net/URL;->openConnection(",
	}
	libraryPrefixes = []string{
		"okhttp3.",
		"retrofit2.",
		"java.net.",
		"javax.net.",
		"com.android.okhttp.",
	}
	reSmaliClass = regexp.MustCompile(`L([^;\s]+);`)
	reJavaClass  = regexp.MustCompile(`\b(?:class|interface|enum)\s+(\w+)`)
	reJavaMethod = regexp.MustCompile(`^(?:(?:public|protected|private|static|final|synchronized|native|abstract|default)\s+)*[\w.<>,\[\]]+\s+(\w+)\s*\([^;]*\)\s*\{?\s*$`)
)

// DecodeDir resolves a package name to .jdt/<pkg>/decode, or returns an existing directory.
func DecodeDir(cwd, arg string) (string, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "", fmt.Errorf("%w: package or decode dir required", ErrUsage)
	}
	st, err := os.Stat(arg)
	if err == nil {
		if !st.IsDir() {
			return "", fmt.Errorf("%w: decode must be a directory", ErrUsage)
		}
		return arg, nil
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("targets: decode: %w", err)
	}
	if isPathArg(arg) {
		return "", fmt.Errorf("targets: decode: %w", err)
	}
	layout, err := workspace.ForPackage(cwd, arg)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrUsage, err.Error())
	}
	st, err = os.Stat(layout.Decode)
	if err != nil {
		return "", fmt.Errorf("targets: decode: %w", err)
	}
	if !st.IsDir() {
		return "", fmt.Errorf("%w: decode must be a directory", ErrUsage)
	}
	return layout.Decode, nil
}

// ScanHTTP walks root for OkHttp, Retrofit, and HttpURLConnection call sites.
func ScanHTTP(root string) (Result, error) {
	st, err := os.Stat(root)
	if err != nil {
		return Result{}, fmt.Errorf("targets: scan: %w", err)
	}
	if !st.IsDir() {
		return Result{}, fmt.Errorf("%w: decode must be a directory", ErrUsage)
	}
	var found []Hit
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		var hits []Hit
		switch {
		case strings.HasSuffix(path, ".smali"):
			hits, err = scanSmali(path)
		case strings.HasSuffix(path, ".java"):
			hits, err = scanJava(path)
		default:
			return nil
		}
		if err != nil {
			return err
		}
		found = append(found, hits...)
		return nil
	})
	if err != nil {
		return Result{}, fmt.Errorf("targets: scan: %w", err)
	}
	return Result{Dir: root, Hits: mergeHits(found)}, nil
}

func isPathArg(arg string) bool {
	if arg == "." || arg == ".." || strings.HasPrefix(arg, ".") || filepath.IsAbs(arg) {
		return true
	}
	return false
}

func scanSmali(path string) ([]Hit, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var className, method string
	var hits []Hit
	for _, line := range strings.Split(string(b), "\n") {
		trim := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trim, ".class "):
			if m := reSmaliClass.FindStringSubmatch(trim); len(m) == 2 {
				className = strings.ReplaceAll(m[1], "/", ".")
			}
			method = ""
		case strings.HasPrefix(trim, ".method "):
			method = smaliMethod(trim)
		case strings.HasPrefix(trim, ".end method"):
			method = ""
		default:
			if method == "" || className == "" || skipClass(className) || !strings.HasPrefix(trim, "invoke-") {
				continue
			}
			if kinds := kindsInInvoke(trim); len(kinds) > 0 {
				hits = append(hits, Hit{Class: className, Method: method, Kinds: kinds})
			}
		}
	}
	return hits, nil
}

func smaliMethod(line string) string {
	i := strings.LastIndex(line, "(")
	if i < 0 {
		return ""
	}
	fields := strings.Fields(line[:i])
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}

func kindsInInvoke(line string) []Kind {
	var kinds []Kind
	if containsAny(line, okhttpSinks) {
		kinds = append(kinds, KindOKHTTP)
	}
	if containsAny(line, retrofitSinks) {
		kinds = append(kinds, KindRetrofit)
	}
	if containsAny(line, urlSinks) {
		kinds = append(kinds, KindURLConnection)
	}
	return kinds
}

func scanJava(path string) ([]Hit, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pkg, className, method string
	methodDepth := 0
	methodOpen := false
	depth := 0
	var hits []Hit
	for _, line := range strings.Split(string(b), "\n") {
		trim := strings.TrimSpace(line)
		if pkg == "" && strings.HasPrefix(trim, "package ") {
			pkg = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trim, "package "), ";"))
		}
		if className == "" {
			if m := reJavaClass.FindStringSubmatch(line); len(m) == 2 {
				className = m[1]
				if pkg != "" {
					className = pkg + "." + className
				}
			}
		}
		if method == "" && className != "" && !skipClass(className) {
			if name, ok := javaMethodName(trim); ok {
				method = name
				methodDepth = depth
				methodOpen = strings.Contains(line, "{")
			}
		}
		if method != "" && !isJavaComment(trim) {
			if kinds := kindsInJava(line); len(kinds) > 0 {
				hits = append(hits, Hit{Class: className, Method: method, Kinds: kinds})
			}
		}
		if method != "" && strings.Contains(line, "{") {
			methodOpen = true
		}
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if method != "" && methodOpen && depth <= methodDepth {
			method = ""
			methodOpen = false
		}
	}
	return hits, nil
}

func javaMethodName(trim string) (string, bool) {
	m := reJavaMethod.FindStringSubmatch(trim)
	if len(m) != 2 {
		return "", false
	}
	switch m[1] {
	case "if", "for", "while", "switch", "catch", "return", "new":
		return "", false
	default:
		return m[1], true
	}
}

func isJavaComment(trim string) bool {
	return strings.HasPrefix(trim, "//") || strings.HasPrefix(trim, "*") || strings.HasPrefix(trim, "/*")
}

func kindsInJava(line string) []Kind {
	var kinds []Kind
	if strings.Contains(line, "okhttp3.") {
		kinds = append(kinds, KindOKHTTP)
	}
	if strings.Contains(line, "retrofit2.") {
		kinds = append(kinds, KindRetrofit)
	}
	if strings.Contains(line, "HttpURLConnection") || strings.Contains(line, "HttpsURLConnection") || strings.Contains(line, "openConnection(") {
		kinds = append(kinds, KindURLConnection)
	}
	return kinds
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func skipClass(class string) bool {
	for _, p := range libraryPrefixes {
		if strings.HasPrefix(class, p) {
			return true
		}
	}
	return false
}

func mergeHits(in []Hit) []Hit {
	type key struct{ class, method string }
	order := make([]key, 0)
	acc := map[key]map[Kind]struct{}{}
	for _, h := range in {
		k := key{h.Class, h.Method}
		set, ok := acc[k]
		if !ok {
			set = map[Kind]struct{}{}
			acc[k] = set
			order = append(order, k)
		}
		for _, kind := range h.Kinds {
			set[kind] = struct{}{}
		}
	}
	hits := make([]Hit, 0, len(order))
	for _, k := range order {
		kinds := make([]Kind, 0, len(acc[k]))
		for kind := range acc[k] {
			kinds = append(kinds, kind)
		}
		sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })
		hits = append(hits, Hit{Class: k.class, Method: k.method, Kinds: kinds})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Class != hits[j].Class {
			return hits[i].Class < hits[j].Class
		}
		return hits[i].Method < hits[j].Method
	})
	return hits
}

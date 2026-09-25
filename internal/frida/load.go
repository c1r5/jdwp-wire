package frida

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// writeLoader emits one script that evals the user sources when the app
// class loader exists. handleBindApplication is already on the stack while
// the process waits for the debugger, so Java.perform's hook on that method
// never sees this launch and never runs after the debugger resumes.
// Application.attach runs later, still before Application.onCreate.
func writeLoader(dir string, files []Ref) (string, error) {
	srcs := make([]string, 0, len(files))
	for _, ref := range files {
		body, err := os.ReadFile(ref.Path)
		if err != nil {
			return "", fmt.Errorf("frida: script: %w", err)
		}
		quoted, err := json.Marshal(string(body))
		if err != nil {
			return "", fmt.Errorf("frida: script: %w", err)
		}
		srcs = append(srcs, string(quoted))
	}
	var b strings.Builder
	b.WriteString(loaderHead)
	for i, src := range srcs {
		if i > 0 {
			b.WriteString(",\n")
		}
		b.WriteString(src)
	}
	b.WriteString(loaderTail)
	path := filepath.Join(dir, "load.js")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("frida: script: %w", err)
	}
	return path, nil
}

const loaderHead = `'use strict';
(function () {
  var sources = [
`

const loaderTail = `
  ];
  var ran = false;
  function runSources() {
    if (ran) {
      return;
    }
    ran = true;
    sources.forEach(function (src) {
      try {
        (0, eval)(src);
      } catch (e) {
        console.error(String(e));
      }
    });
  }
  function arm(loader) {
    if (loader !== null && loader !== undefined) {
      Java.classFactory.loader = loader;
    }
    runSources();
  }
  if (typeof Java === 'undefined' || !Java.available) {
    runSources();
    return;
  }
  var early = null;
  var ready = false;
  Java.performNow(function () {
    var thread = Java.use('android.app.ActivityThread');
    var current = thread.currentApplication();
    if (current !== null) {
      early = current.getClassLoader();
      ready = true;
      return;
    }
    var Application = Java.use('android.app.Application');
    var attach = Application.attach.overload('android.content.Context');
    attach.implementation = function (ctx) {
      arm(ctx.getClassLoader());
      return attach.call(this, ctx);
    };
    console.log('frida: Java hooks wait for Application.attach');
  });
  if (ready) {
    arm(early);
  }
})();
`

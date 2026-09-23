'use strict';

Java.perform(function () {
  try {
    var Debug = Java.use('android.os.Debug');
    Debug.isDebuggerConnected.implementation = function () {
      return false;
    };
  } catch (e) {}
});

function maskTracer(buf, n) {
  try {
    if (!buf || n <= 0 || n > 4096) {
      return;
    }
    var s = buf.readUtf8String(n);
    if (!s || s.indexOf('TracerPid:') === -1) {
      return;
    }
    buf.writeUtf8String(s.replace(/TracerPid:[ \t]*[0-9]+/, 'TracerPid:\t0'));
  } catch (e) {}
}

try {
  var fgets = Module.findExportByName(null, 'fgets');
  if (fgets) {
    Interceptor.attach(fgets, {
      onEnter: function (args) {
        this.buf = args[0];
        this.cap = args[1].toInt32();
      },
      onLeave: function (ret) {
        if (ret.isNull()) {
          return;
        }
        maskTracer(this.buf, this.cap > 0 ? this.cap : 256);
      },
    });
  }
} catch (e) {}

try {
  var readp = Module.findExportByName(null, 'read');
  if (readp) {
    Interceptor.attach(readp, {
      onEnter: function (args) {
        this.buf = args[1];
      },
      onLeave: function (ret) {
        maskTracer(this.buf, ret.toInt32());
      },
    });
  }
} catch (e) {}

console.log('antidebug-bypass armed');

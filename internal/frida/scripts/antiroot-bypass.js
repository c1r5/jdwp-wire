'use strict';

Java.perform(function () {
  function guard(fn) {
    try {
      fn();
    } catch (e) {}
  }

  var exact = {
    '/system/bin/su': true,
    '/system/xbin/su': true,
    '/sbin/su': true,
    '/su/bin/su': true,
    '/data/local/su': true,
    '/data/local/bin/su': true,
    '/data/local/xbin/su': true,
    '/system/bin/magisk': true,
    '/sbin/magisk': true,
    '/data/adb/magisk': true,
  };

  function looksRoot(p) {
    if (!p) {
      return false;
    }
    var s = '' + p;
    if (exact[s]) {
      return true;
    }
    if (s.indexOf('/magisk') !== -1 || s.indexOf('supersu') !== -1) {
      return true;
    }
    return s === '/su' || s.slice(-3) === '/su';
  }

  function isSuCmd(v) {
    var text = '';
    try {
      if (v && v.length !== undefined && typeof v !== 'string') {
        var parts = [];
        for (var i = 0; i < v.length; i++) {
          parts.push('' + v[i]);
        }
        text = parts.join(' ');
      } else {
        text = '' + v;
      }
    } catch (e) {
      return false;
    }
    var t = text.trim();
    return t === 'su' || t.indexOf('su ') === 0 || t.indexOf('/su') !== -1 || t.indexOf('which su') !== -1 || t.indexOf('magisk') !== -1;
  }

  guard(function () {
    var File = Java.use('java.io.File');
    var exists = File.exists;
    exists.implementation = function () {
      try {
        if (looksRoot(this.getAbsolutePath())) {
          return false;
        }
      } catch (e) {}
      return exists.call(this);
    };
  });

  guard(function () {
    var RB = Java.use('com.scottyab.rootbeer.RootBeer');
    [
      'isRooted',
      'isRootedWithoutBusyBoxCheck',
      'detectRootManagementApps',
      'detectPotentiallyDangerousApps',
      'detectTestKeys',
      'checkForBinary',
      'checkForDangerousProps',
      'checkForRWPaths',
      'detectRootCloakingApps',
      'checkSuExists',
      'checkForRootNative',
      'checkForMagiskBinary',
    ].forEach(function (name) {
      try {
        RB[name].overloads.forEach(function (m) {
          m.implementation = function () {
            return false;
          };
        });
      } catch (e) {}
    });
  });

  guard(function () {
    var Build = Java.use('android.os.Build');
    var tags = Build.TAGS.value;
    if (tags && ('' + tags).indexOf('test-keys') !== -1) {
      Build.TAGS.value = ('' + tags).replace('test-keys', 'release-keys');
    }
  });

  guard(function () {
    var Runtime = Java.use('java.lang.Runtime');
    var IO = Java.use('java.io.IOException');
    Runtime.exec.overloads.forEach(function (m) {
      m.implementation = function () {
        try {
          if (isSuCmd(arguments[0])) {
            throw IO.$new('su');
          }
        } catch (e) {
          if (('' + e).indexOf('su') !== -1) {
            throw e;
          }
        }
        return m.call.apply(m, [this].concat([].slice.call(arguments)));
      };
    });
  });

  console.log('antiroot-bypass armed');
});

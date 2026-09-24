'use strict';

Java.perform(function () {
  function guard(fn) {
    try {
      fn();
    } catch (e) {}
  }

  guard(function () {
    var X509TrustManager = Java.use('javax.net.ssl.X509TrustManager');
    var SSLContext = Java.use('javax.net.ssl.SSLContext');
    var TrustAll = Java.registerClass({
      name: 'com.jdt.frida.TrustAll',
      implements: [X509TrustManager],
      methods: {
        checkClientTrusted: function () {},
        checkServerTrusted: function () {},
        getAcceptedIssuers: function () {
          return [];
        },
      },
    });
    var managers = [TrustAll.$new()];
    var init = SSLContext.init.overload(
      '[Ljavax.net.ssl.KeyManager;',
      '[Ljavax.net.ssl.TrustManager;',
      'java.security.SecureRandom'
    );
    init.implementation = function (km, tm, sr) {
      return init.call(this, km, managers, sr);
    };
  });

  guard(function () {
    var Pinner = Java.use('okhttp3.CertificatePinner');
    Pinner.check.overloads.forEach(function (m) {
      m.implementation = function () {};
    });
  });

  guard(function () {
    var OkHV = Java.use('okhttp3.internal.tls.OkHostnameVerifier');
    OkHV.verify.overloads.forEach(function (m) {
      m.implementation = function () {
        return true;
      };
    });
  });

  guard(function () {
    var HV = Java.use('javax.net.ssl.HostnameVerifier');
    var Allow = Java.registerClass({
      name: 'com.jdt.frida.AllowHost',
      implements: [HV],
      methods: {
        verify: function () {
          return true;
        },
      },
    });
    var allow = Allow.$new();
    var HTTPS = Java.use('javax.net.ssl.HttpsURLConnection');
    HTTPS.setDefaultHostnameVerifier.implementation = function () {
      return HTTPS.setDefaultHostnameVerifier.call(this, allow);
    };
    HTTPS.setHostnameVerifier.implementation = function () {
      return HTTPS.setHostnameVerifier.call(this, allow);
    };
  });

  guard(function () {
    var TM = Java.use('com.android.org.conscrypt.TrustManagerImpl');
    ['verifyChain', 'checkTrustedRecursive'].forEach(function (name) {
      try {
        TM[name].overloads.forEach(function (m) {
          m.implementation = function () {
            return arguments[0];
          };
        });
      } catch (e) {}
    });
  });

  guard(function () {
    var Handler = Java.use('android.webkit.SslErrorHandler');
    Handler.cancel.implementation = function () {
      try {
        this.proceed();
      } catch (e) {}
    };
  });

  console.log('sslpinning-bypass armed');
});

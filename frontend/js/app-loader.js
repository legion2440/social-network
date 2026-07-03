(function () {
  var scripts = document.querySelectorAll('script[data-dc-script][data-src]');
  for (var i = 0; i < scripts.length; i += 1) {
    var script = scripts[i];
    var src = script.getAttribute('data-src');
    if (!src) continue;
    var xhr = new XMLHttpRequest();
    xhr.open('GET', src, false);
    xhr.send(null);
    if ((xhr.status >= 200 && xhr.status < 300) || xhr.status === 0) {
      script.textContent = xhr.responseText;
    } else {
      throw new Error('Failed to load ' + src + ': HTTP ' + xhr.status);
    }
  }
})();

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const { execFileSync } = require('node:child_process');

const frontendDir = path.resolve(__dirname, '..');
const distDir = path.join(frontendDir, 'dist');

function build() {
  execFileSync(process.execPath, ['scripts/build.mjs'], {
    cwd: frontendDir,
    stdio: 'pipe'
  });
}

function snapshotTree(directory, relative = '') {
  return fs.readdirSync(path.join(directory, relative), { withFileTypes: true })
    .sort((left, right) => left.name.localeCompare(right.name))
    .flatMap(entry => {
      const entryPath = path.join(relative, entry.name);
      if (entry.isDirectory()) return snapshotTree(directory, entryPath);
      const content = fs.readFileSync(path.join(directory, entryPath));
      return [{
        path: entryPath.split(path.sep).join('/'),
        size: content.length,
        sha256: crypto.createHash('sha256').update(content).digest('hex')
      }];
    });
}

test('clean production build embeds app source and emits only runtime assets', () => {
  fs.rmSync(distDir, { recursive: true, force: true });
  build();

  const html = fs.readFileSync(path.join(distDir, 'index.html'), 'utf8');
  assert.match(html, /<script\b[^>]*type="text\/x-dc"[^>]*data-dc-script=""[^>]*>\s*const/);
  assert.doesNotMatch(html, /data-src=/);
  assert.doesNotMatch(html, /app-loader\.js|resources\.js|XMLHttpRequest/);
  assert.doesNotMatch(html, /<!-- @include /);
  assert.doesNotMatch(html, /<script[^>]+babel\.min\.js/);
  assert.doesNotMatch(html, /(?:src|href)="(?:js|css|assets)\//);
  assert.equal(fs.existsSync(path.join(distDir, 'js', 'app.js')), false);
  assert.equal(fs.existsSync(path.join(distDir, 'js', 'app-loader.js')), false);
  assert.equal(fs.existsSync(path.join(distDir, 'js', 'resources.js')), false);
  assert.equal(fs.existsSync(path.join(distDir, 'js', 'vendor', 'babel.min.js')), true);

  const head = html.match(/<head>([\s\S]*?)<\/head>/i);
  assert.ok(head);
  const firstMeaningful = head[1].replace(/<!--[\s\S]*?-->/g, '').trimStart();
  assert.match(firstMeaningful, /^<meta charset="utf-8">/i);
});

test('production runtime and CSS use root-absolute asset URLs', () => {
  build();

  const runtime = fs.readFileSync(path.join(distDir, 'js', 'runtime.js'), 'utf8');
  assert.match(runtime, /"\/js\/vendor\/babel\.min\.js"/);
  assert.match(runtime, /"\/js\/vendor\/react\.production\.min\.js"/);
  assert.doesNotMatch(runtime, /"js\/vendor\//);

  const css = fs.readFileSync(path.join(distDir, 'css', 'base.css'), 'utf8');
  assert.doesNotMatch(css, /url\("\.\.\/assets\//);
  assert.match(css, /url\("\/assets\//);
});

test('repeated clean builds produce the same output tree', () => {
  build();
  const first = snapshotTree(distDir);
  fs.writeFileSync(path.join(distDir, 'stale-output.txt'), 'must be removed\n', 'utf8');

  build();
  assert.deepEqual(snapshotTree(distDir), first);
  assert.equal(fs.existsSync(path.join(distDir, 'stale-output.txt')), false);
});

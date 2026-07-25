import { cp, mkdir, readFile, readdir, rm, writeFile } from 'node:fs/promises';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const frontendDir = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const sourceDir = join(frontendDir, 'src');
const outputDir = join(frontendDir, 'dist');
const templateDir = join(sourceDir, 'templates');

async function expandTemplateIncludes(source) {
  const includePattern = /<!-- @include ([a-z0-9-]+\.html) -->/;
  let expanded = source;
  for (;;) {
    const match = expanded.match(includePattern);
    if (!match) return expanded;
    const fragment = await readFile(join(templateDir, match[1]), 'utf8');
    expanded = expanded.replace(match[0], fragment.trim());
  }
}

await rm(outputDir, { recursive: true, force: true });
await mkdir(outputDir, { recursive: true });

await cp(join(sourceDir, 'assets'), join(outputDir, 'assets'), { recursive: true });
await cp(join(sourceDir, 'css'), join(outputDir, 'css'), { recursive: true });
await cp(join(sourceDir, 'js'), join(outputDir, 'js'), { recursive: true });

await rm(join(outputDir, 'js', 'app.js'), { force: true });

const appSource = await readFile(join(sourceDir, 'js', 'app.js'), 'utf8');
let html = await readFile(join(sourceDir, 'index.html'), 'utf8');
html = await expandTemplateIncludes(html);
const scriptMarker = /(<script\b[^>]*\btype="text\/x-dc"[^>]*\bdata-dc-script=""[^>]*)\sdata-src="\/js\/app\.js"([^>]*>)\s*<\/script>/;
if (!scriptMarker.test(html)) {
  throw new Error('source index is missing the dc-runtime application script marker');
}
html = html.replace(scriptMarker, (_, before, after) => {
  const embedded = appSource.replaceAll('</script', '<\\/script');
  return before + after + '\n' + embedded + '\n</script>';
});
await writeFile(join(outputDir, 'index.html'), html, 'utf8');

const runtimePath = join(outputDir, 'js', 'runtime.js');
const runtime = (await readFile(runtimePath, 'utf8')).replaceAll('"js/vendor/', '"/js/vendor/');
await writeFile(runtimePath, runtime, 'utf8');

for (const stylesheet of (await readdir(join(outputDir, 'css'))).filter(name => name.endsWith('.css'))) {
  const path = join(outputDir, 'css', stylesheet);
  const css = (await readFile(path, 'utf8')).replaceAll('url("../assets/', 'url("/assets/');
  await writeFile(path, css, 'utf8');
}

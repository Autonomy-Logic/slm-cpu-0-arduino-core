#!/usr/bin/env node
// merge-components.mjs — merge the hand-maintained inventory into a cdxgen BOM.
//   node scripts/merge-components.mjs <cdx.json> <components.json> <upstream.json> [version]
// Env: GO_VERSION (e.g. go1.22.12) pins the Go stdlib version; without it the go.mod directive is used.
// Rewrites <cdx.json> in place. Exit 1 when the inventory disagrees with upstream.json.

import { readFileSync, writeFileSync } from 'node:fs';
import { refOf } from './lib/cdx.mjs';

const [cdxPath, compsPath, upstreamPath, versionArg = ''] = process.argv.slice(2);
if (!cdxPath || !compsPath || !upstreamPath) {
  console.error('usage: merge-components.mjs <cdx.json> <components.json> <upstream.json> [version]');
  process.exit(1);
}

const fail = (msg) => { console.error(`::error::merge-components: ${msg}`); process.exit(1); };
const readJson = (p) => {
  try { return JSON.parse(readFileSync(p, 'utf8')); } catch (e) { return fail(`cannot read/parse "${p}" (${e.code || e.message})`); }
};

const cdx = readJson(cdxPath);
const inv = readJson(compsPath);
const upstream = readJson(upstreamPath);
if (!inv.root?.name) fail(`"${compsPath}" has no root.name`);
if (!Array.isArray(inv.components)) fail(`"${compsPath}" has no components array`);

const version = versionArg.replace(/^v/, '');

// Go stdlib version: the toolchain that built the binary, else the go.mod minimum.
function goVersion(c) {
  const env = (process.env.GO_VERSION || '').trim().replace(/^go/, '');
  if (env) return env;
  if (!c.goMod) fail(`component "${c.name}" uses versionFrom=go but has no goMod`);
  const m = readFileSync(c.goMod, 'utf8').match(/^go\s+(\S+)/m);
  if (!m) fail(`no go directive in ${c.goMod}`);
  return m[1];
}

// Inventory entries must match tools/ci/upstream.json, and every upstream entry must be inventoried.
const errors = [];
const covered = { archive: new Set(), tool: new Set() };
for (const c of inv.components) {
  const up = c.upstream;
  if (!up) continue;
  if (up.archive) {
    const a = upstream.archives?.[up.archive];
    covered.archive.add(up.archive);
    if (!a) { errors.push(`${c.name}: archive "${up.archive}" is not in upstream.json`); continue; }
    if (!String(a.url).includes(`${c.version}`)) errors.push(`${c.name}: version ${c.version} does not match ${a.url}`);
    c.hashes = [{ alg: 'SHA-256', content: a.sha256 }];
    c.externalReferences = [{ type: 'distribution', url: a.url }];
  } else if (up.tool) {
    const t = (upstream.tools || []).find((x) => x.name === up.tool);
    covered.tool.add(up.tool);
    if (!t) { errors.push(`${c.name}: tool "${up.tool}" is not in upstream.json`); continue; }
    if (t.version !== c.version) errors.push(`${c.name}: version ${c.version} != upstream.json ${t.version}`);
    const linux = (t.systems || []).find((s) => s.host === 'x86_64-pc-linux-gnu') || t.systems?.[0];
    if (linux?.url) c.externalReferences = [{ type: 'distribution', url: linux.url }];
  }
}
for (const k of Object.keys(upstream.archives || {})) if (!covered.archive.has(k)) errors.push(`upstream archive "${k}" has no component in ${compsPath}`);
for (const t of upstream.tools || []) if (!covered.tool.has(t.name)) errors.push(`upstream tool "${t.name}" has no component in ${compsPath}`);
if (errors.length) fail(`inventory out of date:\n  - ${errors.join('\n  - ')}`);

// CycloneDX supplier is an object; the inventory may give a plain name.
const supplierOf = (c) => {
  const name = typeof c.supplier === 'string' ? c.supplier : c.supplier?.name;
  return name ? { supplier: { name } } : {};
};

const toComponent = (c) => {
  const v = c.versionFrom === 'go' ? goVersion(c) : c.versionFrom === 'root' ? version : c.version;
  const purl = c.purl && v && !c.purl.includes('@') ? `${c.purl}@${v}` : c.purl;
  const out = {
    type: c.type || 'library', name: c.name, ...(c.group ? { group: c.group } : {}),
    ...(v ? { version: v } : {}), ...supplierOf(c), ...(c.publisher ? { publisher: c.publisher } : {}),
    ...(c.description ? { description: c.description } : {}),
    ...(purl ? { purl, 'bom-ref': purl } : {}),
    licenses: c.licenses || [],
    ...(c.hashes ? { hashes: c.hashes } : {}),
    ...(c.externalReferences ? { externalReferences: c.externalReferences } : {}),
    properties: [
      { name: 'autonomy:inventory', value: 'security/components.json' },
      ...(c.distribution ? [{ name: 'autonomy:distribution', value: c.distribution }] : []),
    ],
  };
  if (!out['bom-ref']) out['bom-ref'] = refOf(out);
  return out;
};

// purl without version identifies the same package across cdxgen and the inventory.
const purlKey = (p) => String(p || '').split('@')[0].toLowerCase();

const generated = [...(cdx.components || [])];
// When cdxgen makes the Go module the root, it becomes an ordinary component under the platform root.
const cdxRoot = cdx.metadata?.component;
if (cdxRoot && /^pkg:golang\//.test(cdxRoot.purl || '')) generated.push(cdxRoot);

const manual = inv.components.map(toComponent);
const byKey = new Map();
const remap = new Map();
for (const c of generated) byKey.set(purlKey(c.purl) || refOf(c), c);
for (const m of manual) {
  const k = purlKey(m.purl) || refOf(m);
  const g = byKey.get(k);
  if (g) remap.set(refOf(g), refOf(m));
  // The inventory is authoritative for identity and licence; cdxgen keeps anything else it found.
  byKey.set(k, g ? { ...g, ...m, properties: [...(g.properties || []), ...m.properties] } : m);
}
const ref = (r) => remap.get(r) || r;
const components = [...byKey.values()].map((c) => {
  const { components: _nested, ...rest } = c;
  return rest;
});

const rootPurl = version ? `${inv.root.purl}@${version}` : inv.root.purl;
const root = {
  type: inv.root.type || 'application', name: inv.root.name, ...(version ? { version } : {}),
  ...supplierOf(inv.root), ...(inv.root.publisher ? { publisher: inv.root.publisher } : {}),
  ...(inv.root.description ? { description: inv.root.description } : {}),
  purl: rootPurl, 'bom-ref': rootPurl, licenses: inv.root.licenses || [],
};

cdx.metadata = { ...(cdx.metadata || {}), component: root };
cdx.components = components;
const known = new Set(components.map(refOf));
const deps = (cdx.dependencies || []).map((d) => ({ ref: ref(d.ref), dependsOn: (d.dependsOn || []).map(ref).filter((r) => known.has(r)) }))
  .filter((d) => known.has(d.ref));
const depMap = new Map(components.map((c) => [refOf(c), new Set()]));
for (const d of deps) for (const r of d.dependsOn) depMap.get(d.ref).add(r);
depMap.set(root['bom-ref'], new Set(components.map(refOf)));
cdx.dependencies = [...depMap].map(([r, on]) => ({ ref: r, dependsOn: [...on] }));

writeFileSync(cdxPath, JSON.stringify(cdx, null, 2));
console.log(`Merged ${manual.length} inventory component(s); BOM now has ${components.length} component(s) under "${root.name}"${version ? `@${version}` : ''}.`);

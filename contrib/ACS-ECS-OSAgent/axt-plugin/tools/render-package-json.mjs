// Renders axt-plugin/package.json from package.json.in by extracting:
//   - {{ VERSION }}      ← packages/cli/package.json#version
//   - {{ DEPENDENCIES }} ← `external` list from `node esbuild.config.mjs --print-config`,
//                          cross-referenced against packages/cli/package.json#dependencies
//                          for version pins.
//
// The CLI's esbuild.config.mjs is a script (esbuild's intentional design); we
// invoke it in `--print-config` mode to obtain the resolved config object as
// JSON, so the runtime dependency list is never duplicated here.
//
// Usage:
//   node tools/render-package-json.mjs [outputPath]
// The output path defaults to <plugin>/build/_shared/package.json so the
// source tree is never mutated by the build.
import { execFileSync } from "node:child_process";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const PLUGIN_DIR = resolve(__dirname, "..");
const CLI_DIR = resolve(PLUGIN_DIR, "../packages/cli");
const DEFAULT_OUT = resolve(PLUGIN_DIR, "build/_shared/package.json");
const outPath = resolve(process.cwd(), process.argv[2] ?? DEFAULT_OUT);

const cliPkg = JSON.parse(readFileSync(resolve(CLI_DIR, "package.json"), "utf-8"));

// Invoke esbuild.config.mjs in print-config mode to obtain the resolved config.
const stdout = execFileSync(
  process.execPath,
  [resolve(CLI_DIR, "esbuild.config.mjs"), "--print-config"],
  { cwd: CLI_DIR, encoding: "utf-8" },
);
const esbuildConfig = JSON.parse(stdout);
const externals = esbuildConfig.external ?? [];

// Reduce externals (which may contain subpaths like "react/jsx-runtime") to
// their owning package name.
const packageNameOf = (spec) =>
  spec.startsWith("@") ? spec.split("/", 2).join("/") : spec.split("/", 1)[0];

const depNames = [...new Set(externals.map(packageNameOf))].sort();
const deps = {};
for (const name of depNames) {
  const version = cliPkg.dependencies?.[name];
  if (!version) {
    throw new Error(
      `External "${name}" listed in esbuild config but missing from packages/cli/package.json#dependencies`,
    );
  }
  deps[name] = version;
}

// Pretty-print and re-indent so the block nests correctly inside the parent
// JSON object (the placeholder sits at 2-space indentation in the template).
const depsJson = JSON.stringify(deps, null, 2)
  .split("\n")
  .map((line, i) => (i === 0 ? line : "  " + line))
  .join("\n");

const template = readFileSync(resolve(PLUGIN_DIR, "package.json.in"), "utf-8");
const rendered = template
  .replace(/{{\s*VERSION\s*}}/g, cliPkg.version)
  .replace(/{{\s*DEPENDENCIES\s*}}/g, depsJson);

// Fail fast on malformed output rather than shipping a broken manifest.
JSON.parse(rendered);

mkdirSync(dirname(outPath), { recursive: true });
writeFileSync(outPath, rendered);
console.log(
  `Rendered ${outPath} (agent ${cliPkg.version}, ${depNames.length} runtime deps from esbuild config)`,
);

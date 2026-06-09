import * as esbuild from "esbuild";
import { cpSync, readFileSync } from "node:fs";
import { resolve, dirname, basename } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));

// Read version from package.json (Single Source of Truth)
const pkg = JSON.parse(readFileSync(resolve(__dirname, "package.json"), "utf-8"));

/** @type {esbuild.BuildOptions} */
const config = {
  entryPoints: ["src/index.ts"],
  bundle: true,
  platform: "node",
  target: "node20",
  format: "esm",
  outfile: "dist/cli.js",
  sourcemap: true,
  // Enable JSX transform for .tsx files
  jsx: "automatic",
  banner: {
    js: "#!/usr/bin/env node",
  },
  // Inject version at build time
  define: {
    __VERSION__: JSON.stringify(pkg.version),
  },
  // Loader for .tsx files
  loader: {
    ".tsx": "tsx",
    ".ts": "ts",
  },
  // Third-party dependencies to keep as external (runtime dependencies)
  // These are NOT bundled and must be installed by the user
  external: [
    "@agentclientprotocol/sdk",
    "@alicloud/ecs20140526",
    "@alicloud/openapi-client",
    "@alicloud/tea-util",
    "@modelcontextprotocol/sdk",
    "ali-oss",
    "chalk",
    "cli-highlight",
    "cli-spinners",
    "commander",
    "handlebars",
    "https-proxy-agent",
    "ink",
    "js-yaml",
    "json-merge-patch",
    "marked",
    "openai",
    "react",
    "react/jsx-runtime",
    "uuidv7",
    "zod",
  ],
};

// `--print-config` mode: emit the resolved esbuild config as JSON on stdout
// and exit. This lets downstream tooling (e.g., axt-plugin/tools/render-
// package-json.mjs) consume the `external` list without duplicating it.
// Nothing else is allowed to write to stdout in this mode.
if (process.argv.includes("--print-config")) {
  process.stdout.write(JSON.stringify(config, null, 2) + "\n");
  process.exit(0);
}

await esbuild.build(config);
console.log("Build complete: dist/cli.js");

// Copy builtin skills into dist (excluding package.json, .turbo, and src/)
cpSync(
  resolve(__dirname, "../skills"),
  resolve(__dirname, "dist/skills"),
  {
    recursive: true,
    filter: (src) => {
      const name = basename(src);
      // Exclude package.json and .turbo
      if (name === "package.json" || name === ".turbo") {
        return false;
      }
      // Exclude src/ directories and their contents
      if (src.includes("/src/") || name === "src") {
        return false;
      }
      // Exclude source map files
      if (name.endsWith(".js.map")) {
        return false;
      }
      return true;
    },
  },
);
console.log("Copied builtin skills to dist/skills");

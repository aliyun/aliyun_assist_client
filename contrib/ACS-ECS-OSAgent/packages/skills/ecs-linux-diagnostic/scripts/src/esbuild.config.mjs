import * as esbuild from "esbuild";
import { dirname } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const scriptsDir = dirname(__dirname); // parent of src/

/** @type {esbuild.BuildOptions} */
const config = {
  entryPoints: [`${__dirname}/diagnostic-cli.ts`],
  bundle: true,
  platform: "node",
  target: "node20",
  format: "esm",
  outfile: `${scriptsDir}/ecs-diagnostic-cli.js`,
  sourcemap: true,
  // Keep Alibaba Cloud and OSS SDKs as external runtime dependencies
  external: [
    "@alicloud/ecs20140526",
    "@alicloud/openapi-client",
    "@alicloud/tea-util",
    "ali-oss",
  ],
};

await esbuild.build(config);
console.log("Build complete: scripts/ecs-diagnostic-cli.js");

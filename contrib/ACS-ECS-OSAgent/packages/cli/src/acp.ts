import { existsSync, statSync } from "node:fs";
import { isAbsolute, resolve } from "node:path";

import { Command } from "commander";
import { AppServer, loadConfig, resolveModelConfig, formatTurnError } from "core";

import { startAcpServer } from "./acp/index.js";

// ---------------------------------------------------------------------------
// Helpers (mirror the logic in index.ts — kept local to avoid circular deps)
// ---------------------------------------------------------------------------

function resolveCwd(paths: string[] | undefined): string {
  if (!paths || paths.length === 0) {
    return process.cwd();
  }

  let cwd = process.cwd();
  for (const p of paths) {
    if (p === "") {
      continue;
    }
    cwd = isAbsolute(p) ? p : resolve(cwd, p);
  }

  if (!existsSync(cwd)) {
    throw new Error(`-C path does not exist: ${cwd}`);
  }
  if (!statSync(cwd).isDirectory()) {
    throw new Error(`-C path is not a directory: ${cwd}`);
  }
  return cwd;
}

// ---------------------------------------------------------------------------
// Subcommand factory
// ---------------------------------------------------------------------------

export function makeAcpCommand(): Command {
  const cmd = new Command("acp")
    .description("Start the ACP (Agent Communication Protocol) stdio server")
    .option("--profile <profile>", "Agent profile: 'default' or 'leaf-node' (env: OSAGENT_PROFILE)")
    .action(async function (this: Command) {
      const opts = this.optsWithGlobals();
      const resolvedCwd = resolveCwd(opts.C as string[] | undefined);
      const config = loadConfig(resolvedCwd);

      const profile = opts.profile ?? config.profile;
      if (profile !== "default" && profile !== "leaf-node") {
        throw new Error(`Invalid profile "${profile}". Valid values: default, leaf-node`);
      }

      const modelResult = resolveModelConfig(config);
      if (!modelResult.ok) {
        throw new Error(modelResult.error);
      }
      const modelConfig = modelResult.config;

      const server = new AppServer(config, modelConfig, profile);
      try {
        await startAcpServer(server);
      } catch (error) {
        for (const line of formatTurnError(error)) {
          console.error(line);
        }
        process.exitCode = 1;
      } finally {
        try {
          await server.dispose();
        } catch {
          // Best-effort cleanup
        }
      }
    });

  return cmd;
}

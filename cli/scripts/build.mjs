import * as esbuild from "esbuild";
import { mkdirSync, rmSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const cliRoot = path.join(__dirname, "..");
const constraintsEntry = path.join(cliRoot, "../shared/constraints/index.ts");
const distDir = path.join(cliRoot, "dist");

rmSync(distDir, { recursive: true, force: true });
mkdirSync(distDir, { recursive: true });

const buildOptions = {
  entryPoints: [path.join(cliRoot, "src/index.ts")],
  bundle: true,
  platform: "node",
  target: "node18",
  format: "esm",
  packages: "external",
  outfile: path.join(distDir, "index.js"),
  alias: {
    "@permission-slip/constraints-format": constraintsEntry,
  },
  sourcemap: true,
  logLevel: "info",
};

if (process.argv.includes("--watch")) {
  const ctx = await esbuild.context(buildOptions);
  await ctx.watch();
  console.log("Watching for changes...");
} else {
  await esbuild.build(buildOptions);
}

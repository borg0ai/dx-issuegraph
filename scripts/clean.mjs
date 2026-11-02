#!/usr/bin/env node
// Remove release and docs build output (run `pnpm clean` after a release).
// Only gitignored paths: build/ (docs pin cache), dist/ (scripts/build-release.sh
// archives), apps/cli/bin/ (pnpm build:go output).
import { rmSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");

for (const dir of ["build", "dist", "apps/cli/bin"]) {
  rmSync(join(root, dir), { recursive: true, force: true });
  console.log(`removed ${dir}`);
}

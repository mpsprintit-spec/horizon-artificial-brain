import { cp, mkdir, rm } from "node:fs/promises";
import { spawnSync } from "node:child_process";

await rm("dist", { recursive: true, force: true });
const result = spawnSync("tsc", [], { stdio: "inherit" });
if (result.status !== 0) process.exit(result.status ?? 1);

await mkdir("dist/lib", { recursive: true });
await cp("dist/lib/horizon-store.js", "lib/horizon-store.js");
await cp("index.html", "dist/index.html");
await cp("styles.css", "dist/styles.css");
await cp("app.js", "dist/app.js");

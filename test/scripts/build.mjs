import { cp, mkdir, rm, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const thisDir = path.dirname(fileURLToPath(import.meta.url));
const rootDir = path.resolve(thisDir, "..");
const srcDir = path.join(rootDir, "src");
const buildDir = path.join(rootDir, "build");
const distDir = path.join(buildDir, "dist");

const cleanOnly = process.argv.includes("--clean");

await rm(buildDir, { recursive: true, force: true });
if (cleanOnly) {
  console.log("cleaned", buildDir);
  process.exit(0);
}

await mkdir(distDir, { recursive: true });
await cp(srcDir, distDir, { recursive: true });

const payload = {
  generatedAt: new Date().toISOString(),
  rows: Array.from({ length: 1200 }, (_, i) => ({
    id: i + 1,
    slug: `item-${i + 1}`,
    text: "spack-spa-compression-demo-content",
  })),
};

const payloadPath = path.join(distDir, "assets", "payload.json");
await mkdir(path.dirname(payloadPath), { recursive: true });
await writeFile(payloadPath, JSON.stringify(payload, null, 2), "utf8");

console.log("build complete:", distDir);

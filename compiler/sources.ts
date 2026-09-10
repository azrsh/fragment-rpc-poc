import { readdir } from "node:fs/promises";
import { readFileSync, existsSync } from "node:fs";
import { join, resolve } from "node:path";
import { execFileSync } from "node:child_process";
import type { SourceFile } from "./compile.js";

export async function collectSources(roots = ["app", "mobile/ios/Sources", "mobile/android/app/src/main"]): Promise<SourceFile[]> {
  const files: string[] = [];
  async function walk(dir: string) {
    if (!existsSync(dir)) return;
    for (const e of (await readdir(dir, { withFileTypes: true })).sort((a,b) => a.name.localeCompare(b.name))) {
      const path = join(dir, e.name);
      if (e.isDirectory()) await walk(path);
      else files.push(path);
    }
  }
  for (const root of roots) await walk(root);
  const sources = files.filter(p => p.endsWith(".graphql")).map(name => ({ name, body: readFileSync(name, "utf8") }));
  for (const [extension, executable] of [
    [".swift", "mobile/swift-graphql/.build/debug/extract-graphql"],
    [".kt", "mobile/android/graphql-tools/build/install/graphql-tools/bin/graphql-tools"],
  ]) {
    const native = files.filter(p => p.endsWith(extension));
    if (!native.length) continue;
    if (!existsSync(executable)) throw new Error("Run bash scripts/build-native-tools.sh before generating native contracts");
    sources.push(...JSON.parse(execFileSync(resolve(executable), native, {
      encoding: "utf8", env: { ...process.env, JAVA_HOME: process.env.JAVA_HOME ?? resolve(".local/mobile-tools/jdk/Contents/Home") },
    })) as SourceFile[]);
  }
  return sources;
}

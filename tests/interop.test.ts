import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { once } from "node:events";
import { cp, mkdtemp, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { test } from "node:test";
import { createClient, Code, ConnectError } from "@connectrpc/connect";
import { createGrpcTransport } from "@connectrpc/connect-node";
import { createConnectTransport } from "@connectrpc/connect-web";
import { AppService } from "../generated/app_pb.js";

test("existing Connect ES and gRPC clients work with the standalone Go server", { timeout: 15000 }, async t => {
  const cwd = await mkdtemp(join(tmpdir(), "fragment-rpc-standalone-"));
  await writeFile(join(cwd, "index.html"), "<!doctype html><html><body>RPC test</body></html>");
  await cp(resolve("public/avatars"), join(cwd, "avatars"), { recursive: true });
  const server = spawn(resolve(".local/fragment-rpc"), ["-static", cwd], {
    cwd, env: { ...process.env, PORT: "0", GRPC_PORT: "0", PATH: cwd }, stdio: ["ignore", "pipe", "pipe"],
  });
  let log = "";
  let errors = "";
  server.stderr.on("data", chunk => { errors += chunk; });
  const exited = once(server, "exit");
  t.after(async () => {
    server.kill("SIGTERM");
    const [code] = await exited;
    await rm(cwd, { recursive: true, force: true });
    assert.equal(code, 0, errors);
  });
  const urls = await new Promise<{ web: string; grpc: string }>((resolve, reject) => {
    server.on("error", reject);
    server.on("exit", () => reject(new Error(`Server exited before becoming ready: ${errors}`)));
    server.stdout.on("data", chunk => {
      log += chunk;
      const web = /Demo: (http:\/\/\S+)/.exec(log)?.[1];
      const grpc = /Generated gRPC API: (http:\/\/[^/]+)\//.exec(log)?.[1];
      if (web && grpc) resolve({ web, grpc });
    });
  });

  for (const transport of [
    createConnectTransport({ baseUrl: urls.web, useBinaryFormat: true }),
    createGrpcTransport({ baseUrl: urls.grpc }),
  ]) {
    const client = createClient(AppService, transport);
    for (const method of ["getUserPage", "getIosUserPage", "getAndroidUserPage"] as const) {
      const page = await client[method]({ id: "u1" });
      assert.equal(page.user?.name, "Aki Tanaka");
      assert.equal(page.user?.organization?.name, "Northstar Studio");
      assert.equal(Object.hasOwn(page.user!, "email"), false);
      assert.equal(Object.hasOwn(page.user!, "organizationId"), false);
      assert.equal((await client[method]({ id: "u3" })).user?.organization, undefined);
      assert.equal((await client[method]({ id: "missing" })).user, undefined);
      await assert.rejects(client[method]({}), (e: unknown) => e instanceof ConnectError && e.code === Code.InvalidArgument);
    }
    for (const method of ["getUserSummary", "getIosUserSummary", "getAndroidUserSummary"] as const) {
      const summary = await client[method]({ id: "u2" });
      assert.equal(summary.user?.name, "Ren Sato");
      assert.equal(Object.hasOwn(summary.user!, "avatarUrl"), false);
      assert.equal(Object.hasOwn(summary.user!, "organization"), false);
    }
  }
  const home = await fetch(urls.web);
  assert.equal(home.status, 200);
  assert.match(await home.text(), /<html/);
  const avatar = await fetch(`${urls.web}/avatars/aki.svg`);
  assert.equal(avatar.status, 200);
  assert.match(avatar.headers.get("content-type")!, /image\/svg\+xml/);
});

import assert from "node:assert/strict";
import { test } from "node:test";
import { cp, mkdir, mkdtemp, readFile, rm, symlink, writeFile } from "node:fs/promises";
import { join, resolve } from "node:path";
import { tmpdir } from "node:os";
import { pathToFileURL } from "node:url";
import { execFileSync } from "node:child_process";
import { createClient } from "@connectrpc/connect";
import { createGrpcTransport } from "@connectrpc/connect-node";
import { startBackends } from "../server/backends.js";
import { serveGrpc } from "../server/network.js";
import { createResolvers } from "../server/resolvers.js";
import { AppService as OldAppService } from "../generated/app_pb.js";
import { readFileSync } from "node:fs";

test("editing a child fragment changes the generated contract and actual gRPC response", async t => {
  const dir = await mkdtemp(join(tmpdir(), "fragment-rpc-evolution-"));
  t.after(() => rm(dir, { recursive:true, force:true }));
  await Promise.all([
    cp("app", join(dir, "app"), { recursive:true }),
    cp("proto", join(dir, "proto"), { recursive:true }),
    cp("schema.graphql", join(dir, "schema.graphql")),
    mkdir(join(dir, "server")),
    mkdir(join(dir, "generated")),
    symlink(resolve("node_modules"), join(dir, "node_modules"), "dir"),
    writeFile(join(dir, "package.json"), '{"type":"module"}'),
  ]);
  await Promise.all([
    cp("generated/schema.lock.json", join(dir, "generated/schema.lock.json")),
    cp("server/execute.ts", join(dir, "server/execute.ts")),
    cp("server/plan.ts", join(dir, "server/plan.ts")),
  ]);
  const fragment = join(dir, "app/components/UserCard/UserCard.graphql");
  await writeFile(fragment, (await readFile(fragment, "utf8")).replace("  name\n", "  name\n  email\n"));
  execFileSync(process.execPath, ["--import", "tsx", resolve("compiler/generate.ts")], { cwd:dir, stdio:"pipe" });
  const proto = await readFile(join(dir, "generated/app.proto"), "utf8");
  assert.match(proto, /string email = 5;/);
  assert.match(await readFile(join(dir, "generated/fragments.ts"), "utf8"), /email: string/);

  // Runtime only receives generated artifacts and the small plan executor.
  await Promise.all([
    rm(join(dir, "app"), { recursive:true }),
    rm(join(dir, "schema.graphql")),
  ]);
  const { registerOperations } = await import(pathToFileURL(join(dir, "generated/routes.ts")).href);
  const { AppService } = await import(pathToFileURL(join(dir, "generated/app_pb.ts")).href);
  const backends = await startBackends();
  t.after(() => backends.close());
  const gateway = await serveGrpc(router => registerOperations(router, createResolvers(backends.user.url, backends.organization.url)));
  t.after(() => gateway.close());
  const transport = createGrpcTransport({ baseUrl:gateway.url });
  const client = createClient(AppService as typeof OldAppService, transport);
  const response = await client.getUserPage({ id:"u1" });
  assert.equal(Reflect.get(response.user!, "email"), "aki@example.test");
  const oldClient = createClient(OldAppService, transport);
  const oldResponse = await oldClient.getUserPage({ id:"u1" });
  assert.equal(oldResponse.user?.name, "Aki Tanaka");
  assert.equal(oldResponse.user?.organization?.name, "Northstar Studio");
  assert.equal(Object.hasOwn(oldResponse.user!, "email"), false);
});

test("runtime source graph does not import the GraphQL compiler", () => {
  for (const file of ["server/dev.ts", "server/execute.ts", "server/resolvers.ts", "generated/routes.ts", "generated/app_pb.ts"]) {
    assert.doesNotMatch(readFileSync(file, "utf8"), /from ["']graphql["']|from ["'].*compiler\//);
  }
});

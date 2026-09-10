import assert from "node:assert/strict";
import { test } from "node:test";
import { createClient, ConnectError, Code } from "@connectrpc/connect";
import { createGrpcTransport } from "@connectrpc/connect-node";
import { startBackends } from "../server/backends.js";
import { serveGrpc } from "../server/network.js";
import { createResolvers } from "../server/resolvers.js";
import { registerOperations } from "../generated/routes.js";
import { AppService } from "../generated/app_pb.js";
import { execute } from "../server/execute.js";
import { compile } from "../compiler/compile.js";
import { readFile } from "node:fs/promises";

test("real gRPC: generated endpoints join two services and only fetch selected relations", async t => {
  const calls: string[] = [];
  const backends = await startBackends((service,id) => calls.push(`${service}:${id}`));
  t.after(() => backends.close());
  const resolvers = createResolvers(backends.user.url, backends.organization.url);
  const gateway = await serveGrpc(router => registerOperations(router, resolvers));
  t.after(() => gateway.close());
  const client = createClient(AppService, createGrpcTransport({ baseUrl: gateway.url }));

  const page = await client.getUserPage({ id: "u1" });
  assert.equal(page.user?.name, "Aki Tanaka");
  assert.equal(page.user?.organization?.name, "Northstar Studio");
  assert.equal(Object.hasOwn(page.user!, "email"), false);
  assert.equal(Object.hasOwn(page.user!, "organizationId"), false);
  assert.deepEqual(calls, ["UserService.GetUser:u1", "OrganizationService.GetOrganization:o1"]);

  calls.length = 0;
  const summary = await client.getUserSummary({ id: "u2" });
  assert.equal(summary.user?.name, "Ren Sato");
  assert.equal(Object.hasOwn(summary.user!, "avatarUrl"), false);
  assert.equal(Object.hasOwn(summary.user!, "organization"), false);
  assert.deepEqual(calls, ["UserService.GetUser:u2"]);

  calls.length = 0;
  assert.equal((await client.getUserPage({ id: "u3" })).user?.organization, undefined);
  assert.deepEqual(calls, ["UserService.GetUser:u3"]);
  assert.equal((await client.getUserPage({ id: "missing" })).user, undefined);
  calls.length = 0;
  await assert.rejects(client.getUserPage({}), (e: unknown) => e instanceof ConnectError && e.code === Code.InvalidArgument);
  assert.deepEqual(calls, []);

  const schema = await readFile("schema.graphql", "utf8");
  const duplicate = compile(schema, [{ name:"test.graphql", body:'query Duplicate($id:ID!) { first:user(id:$id) { name organization { name } } second:user(id:$id) { id organization { website } } }' }]);
  calls.length = 0;
  await execute(duplicate.operations[0], { id:"u1" }, resolvers, new AbortController().signal);
  assert.deepEqual(calls, ["UserService.GetUser:u1", "OrganizationService.GetOrganization:o1"]);
});

test("runtime validates required output, preserves false/zero and nullable presence", async () => {
  const schema = "type Query { item(enabled:Boolean!, count:Int!): Item } type Item { enabled:Boolean! count:Int! name:String }";
  const plan = compile(schema, [{ name:"test.graphql", body:"query GetItem($enabled:Boolean!, $count:Int!) { item(enabled:$enabled,count:$count) { enabled count name } }" }]).operations[0];
  const signal = new AbortController().signal;
  assert.deepEqual(await execute(plan, { enabled:false, count:0 }, { "Query.item": (_s,args) => args }, signal), { item:{ count:0, enabled:false, name:null } });
  await assert.rejects(execute(plan, { enabled:false, count:0 }, { "Query.item": () => ({ count:0 }) }, signal), /Null for required field/);
  await assert.rejects(execute(plan, { enabled:false, count:1.5 }, { "Query.item": () => ({}) }, signal), /Invalid Int/);
});

test("downstream failure is propagated without blocking unrelated operations", async t => {
  const backends = await startBackends();
  t.after(() => backends.user.close());
  const resolvers = createResolvers(backends.user.url, backends.organization.url);
  const gateway = await serveGrpc(router => registerOperations(router, resolvers));
  t.after(() => gateway.close());
  await backends.organization.close();
  const client = createClient(AppService, createGrpcTransport({ baseUrl: gateway.url }));
  await assert.rejects(client.getUserPage({ id:"u1" }, { timeoutMs:3000 }), (e: unknown) => e instanceof ConnectError && [Code.Unavailable, Code.DeadlineExceeded].includes(e.code));
  assert.equal((await client.getUserSummary({ id:"u1" })).user?.name, "Aki Tanaka");
});

test("cancelling an in-flight RPC aborts the resolver's signal", { timeout: 5000 }, async t => {
  let signalStarted!: () => void;
  let signalCancelled!: () => void;
  const started = new Promise<void>(resolve => { signalStarted = resolve; });
  const cancelled = new Promise<void>(resolve => { signalCancelled = resolve; });
  const gateway = await serveGrpc(router => registerOperations(router, {
    "Query.user": (_source, _args, context) => new Promise((_resolve, reject) => {
      context.signal.addEventListener("abort", () => {
        signalCancelled();
        reject(new ConnectError("Cancelled", Code.Canceled));
      }, { once:true });
      signalStarted();
    }),
  }));
  t.after(() => gateway.close());
  const client = createClient(AppService, createGrpcTransport({ baseUrl:gateway.url }));
  const abort = new AbortController();
  const response = client.getUserPage({ id:"u1" }, { signal:abort.signal });
  const rejected = assert.rejects(response, (e: unknown) => e instanceof ConnectError && e.code === Code.Canceled);
  await started;
  abort.abort();
  await rejected;
  await cancelled;
});

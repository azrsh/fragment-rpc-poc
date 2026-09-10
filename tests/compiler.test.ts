import assert from "node:assert/strict";
import { test } from "node:test";
import { readFile, readdir } from "node:fs/promises";
import { join } from "node:path";
import { compile } from "../compiler/compile.js";

const schema = await readFile("schema.graphql", "utf8");
const sources = await Promise.all((await readdir("app", { recursive: true })).filter(p => p.endsWith(".graphql")).map(async p => ({ name: p, body: await readFile(join("app", p), "utf8") })));
const query = (body: string, sdl = schema) => compile(sdl, [{ name: "test.graphql", body }]);

test("colocated child fragments become dedicated, minimal response contracts", () => {
  const output = compile(schema, sources);
  assert.deepEqual(output.operations.map(o => o.name), ["GetUserPage", "GetUserSummary"]);
  const user = output.operations[0].fields[0];
  assert.deepEqual(user.fields!.map(f => f.name), ["avatarUrl", "id", "name", "organization"]);
  assert.deepEqual(user.fields!.find(f => f.name === "organization")!.fields!.map(f => f.name), ["name", "website"]);
  assert.doesNotMatch(output.proto, /\bemail\b|\borganizationId\b/);
  assert.match(output.proto, /rpc GetUserPage\(GetUserPageRequest\) returns \(GetUserPageResponse\)/);
  assert.match(output.fragmentTypes, /export type Avatar_user = \{ avatarUrl: string;/);
  assert.deepEqual(compile(schema, [...sources].reverse()), output);
});

test("fields from overlapping fragments merge recursively, including aliases", () => {
  const output = query(`
    fragment Base on User { name organization { name } }
    fragment More on User { name organization { website } }
    query GetUser($id: ID!) { profile: user(id: $id) { ...Base ...More label: name } }
  `);
  const user = output.operations[0].fields[0];
  assert.equal(user.name, "user");
  assert.equal(user.responseName, "profile");
  assert.deepEqual(user.fields!.map(f => f.responseName), ["label", "name", "organization"]);
  assert.equal(user.fields!.find(f => f.responseName === "label")!.name, "name");
  assert.deepEqual(user.fields![2].fields!.map(f => f.name), ["name", "website"]);
});

test("invalid fields, missing fragments, cycles and conflicts fail at build time", () => {
  for (const [document, message] of [
    ['query GetUser { user(id: "u1") { typo } }', /Cannot query field "typo"/],
    ['query GetUser { user(id: "u1") { ...Missing } }', /Unknown fragment/],
    ['fragment A on User { ...B } fragment B on User { ...A } query GetUser { user(id:"u1") { ...A } }', /Cannot spread fragment/],
    ['query GetUser { user(id:"u1") { label: name label: email } }', /conflict/],
    ['query GetUser { user(id:$id) { name } }', /Variable "\$id" is not defined/],
  ] as const) assert.throws(() => query(document), message);
});

test("unsupported semantics fail explicitly instead of being silently dropped", () => {
  for (const [document, message] of [
    ['query GetUser($id:ID!, $show:Boolean!) { user(id:$id) { name @include(if:$show) } }', /Directive @include/],
    ['query GetUser($id:ID! = "u1") { user(id:$id) { name } }', /Variable defaults/],
    ['{ user(id:"u1") { name } }', /Only named queries/],
    ['query GetUser { user(id:"u1") { full_name: name } }', /lowerCamelCase/],
  ] as const) assert.throws(() => query(document), message);
  assert.throws(() => query("query List { users { name } }", "type Query { users: [User] } type User { name:String! }"), /Only non-null lists/);
});

test("presence and supported list types are represented in proto", () => {
  const output = query("query List($enabled:Boolean!) { users(enabled:$enabled) { name } }", "type Query { users(enabled:Boolean!): [User!]! } type User { name:String }");
  assert.match(output.proto, /optional bool enabled/);
  assert.match(output.proto, /repeated ListResponse_Users users/);
  assert.match(output.proto, /optional string name/);
});

test("field numbers survive additions; removals reserve names and numbers", () => {
  const initial = query('query GetUser { user(id:"u1") { id name } }');
  const changed = compile(schema, [{ name: "test.graphql", body: 'query GetUser { user(id:"u1") { email id name } }' }], initial.lock);
  const before = initial.lock.messages.GetUserResponse_User.fields;
  const after = changed.lock.messages.GetUserResponse_User.fields;
  assert.equal(after.id.number, before.id.number);
  assert.equal(after.name.number, before.name.number);
  assert.ok(after.email.number > before.name.number);
  const removed = compile(schema, [{ name: "test.graphql", body: 'query GetUser { user(id:"u1") { id name } }' }], changed.lock);
  assert.match(removed.proto, /reserved 3;/);
  assert.match(removed.proto, /reserved "email";/);
  assert.throws(() => compile(schema, [{ name: "test.graphql", body: 'query GetUser { user(id:"u1") { email id name } }' }], removed.lock), /Retired field/);
  assert.throws(() => compile(schema.replace("name: String!", "name: Int!"), [{ name: "test.graphql", body: 'query GetUser { user(id:"u1") { id name } }' }], initial.lock), /Breaking type change/);
});

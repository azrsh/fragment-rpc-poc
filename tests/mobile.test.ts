import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { test } from "node:test";
import { compile } from "../compiler/compile.js";
import { collectSources } from "../compiler/sources.js";
import { nativeModels, swiftModels } from "../compiler/native.js";

const schema = await readFile("schema.graphql", "utf8");
const sources = await collectSources();

test("web, SwiftUI and Compose fragments compile to six distinct RPCs", () => {
  const output = compile(schema, sources);
  assert.deepEqual(output.operations.map(o => o.name), [
    "GetAndroidUserPage", "GetAndroidUserSummary", "GetIosUserPage", "GetIosUserSummary", "GetUserPage", "GetUserSummary",
  ]);
  for (const operation of output.operations) {
    const fields = operation.fields[0].fields!.map(f => f.name);
    assert.deepEqual(fields, operation.name.endsWith("Summary") ? ["id", "name"] : ["avatarUrl", "id", "name", "organization"]);
  }
});

test("editing a native child fragment only changes that platform's contract", () => {
  const original = compile(schema, sources);
  for (const platform of ["ios", "android"]) {
    const changed = compile(schema, sources.map(s => ({ ...s,
      body: s.name.startsWith(`mobile/${platform}/`) && /UserCard\.(swift|kt)$/.test(s.name)
        ? s.body.replace("  name\n", "  name\n  email\n") : s.body,
    })), original.lock);
    for (const operation of changed.operations) {
      const email = operation.fields[0].fields!.some(f => f.name === "email");
      assert.equal(email, operation.name === (platform === "ios" ? "GetIosUserPage" : "GetAndroidUserPage"));
    }
  }
});

test("native fragment types only expose selected fields and map optional relations", () => {
  const output = compile(schema, sources);
  const native = nativeModels(output, sources);
  for (const prefix of ["Ios", "Android"]) {
    const avatar = native.models.find(m => m.name === prefix + "AvatarFragmentData")!;
    assert.deepEqual(avatar.fields.map(f => f.name), ["avatarUrl"]);
    const card = native.models.find(m => m.name === prefix + "UserCardFragmentData")!;
    assert.equal(card.fields.find(f => f.name === "organization")!.optional, true);
    assert(native.conversions.some(c => c.from.name === card.name && c.to.name === avatar.name));
    assert(!native.conversions.some(c => c.from.name === `Get${prefix}UserSummaryResponse_User` && c.to.name === avatar.name));
  }
  assert.match(swiftModels(native), /self.hasOrganization \? .* : nil/);
});

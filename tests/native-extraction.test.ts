import assert from "node:assert/strict";
import { test } from "node:test";
import { mkdtemp, writeFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { collectSources } from "../compiler/sources.js";
import { compile } from "../compiler/compile.js";

test("native parsers ignore comments and extract literal GraphQL with source locations", async t => {
  const dir = await mkdtemp(join(tmpdir(), "native-extraction-"));
  t.after(() => rm(dir, { recursive: true, force: true }));
  await writeFile(join(dir, "Avatar.swift"), `// @GraphQLFragment("fake") enum Fake {}\n@GraphQLFragment("""\n    fragment Avatar_user on User {\n      avatarUrl\n    }\n    """)\nenum AvatarFragment {}\n`);
  await writeFile(join(dir, "Page.kt"), 'package example\n// @GraphQLQuery("fake") object Fake\n@GraphQLQuery("""\nquery Page($' + "{'$'}" + 'id: ID!) { user(id: $' + "{'$'}" + 'id) { ...Avatar_user } }\n""")\nobject PageQuery\n');
  const sources = await collectSources([dir]);
  assert.equal(sources.length, 2);
  assert.equal(sources.find(s => s.native?.language === "swift")!.line, 3);
  assert.match(sources.find(s => s.native?.language === "kotlin")!.body, /query Page\(\$id: ID!\)/);
  const schema = "type Query { user(id: ID!): User } type User { avatarUrl: String! }";
  assert.equal(compile(schema, sources).operations[0].name, "Page");
  assert.throws(() => compile(schema, sources.map(s => ({ ...s, body: s.body.replace("avatarUrl", "unknownField") }))), /Avatar.swift:4:7/);
});

test("native parsers reject dynamically interpolated documents", async t => {
  const dir = await mkdtemp(join(tmpdir(), "native-dynamic-"));
  t.after(() => rm(dir, { recursive: true, force: true }));
  for (const [extension, content] of [
    ["swift", '@GraphQLFragment("""\nfragment F on User { \\(field) }\n""")\nenum F {}'],
    ["kt", '@GraphQLFragment("""\nfragment F on User { $field }\n""")\nobject F'],
  ]) {
    const path = join(dir, `Dynamic.${extension}`);
    await writeFile(path, content);
    await assert.rejects(collectSources([dir]), /interpolation/);
    await rm(path);
  }
});

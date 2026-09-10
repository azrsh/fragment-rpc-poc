import {
  buildSchema, parse, validate, visit, Source, Kind,
  isObjectType, isNonNullType, isListType, getNamedType,
  type DocumentNode, type FieldNode, type FragmentDefinitionNode,
  type GraphQLObjectType, type SelectionSetNode, type GraphQLOutputType,
} from "graphql";
import type { FieldPlan, OperationPlan, Scalar } from "../server/plan.js";

export interface SourceFile {
  name: string;
  body: string;
  line?: number;
  column?: number;
  native?: { language: "swift" | "kotlin"; declaration: string; kind: "fragment" | "query"; packageName?: string };
}
interface LockedField { number: number; signature: string }
export interface ContractLock {
  version: 1;
  messages: Record<string, { fields: Record<string, LockedField>; retired: string[] }>;
}
const scalars: Record<Scalar, string> = {
  ID: "string", String: "string", Int: "int32", Float: "double", Boolean: "bool",
};
const camelName = /^[a-z][a-zA-Z0-9]*$/;
const upper = (s: string) => s[0].toUpperCase() + s.slice(1);
function fail(message: string): never { throw new Error(message); }

export function compile(schemaSDL: string, sources: SourceFile[], previous?: ContractLock) {
  const schema = buildSchema(schemaSDL);
  const sourceOwners = new Map<Source, SourceFile>();
  const document: DocumentNode = {
    kind: Kind.DOCUMENT,
    definitions: sources.flatMap(s => {
      const indentation = s.native?.language === "swift" ? " ".repeat((s.column ?? 1) - 1) : "";
      const body = s.body.split("\n").map((line, i) => i ? indentation + line : line).join("\n");
      const source = new Source(body, s.name, { line: s.line ?? 1, column: s.column ?? 1 });
      sourceOwners.set(source, s);
      const parsed = parse(source);
      if (s.native) {
        const definition = parsed.definitions[0];
        const expected = s.native.kind === "fragment" ? Kind.FRAGMENT_DEFINITION : Kind.OPERATION_DEFINITION;
        if (parsed.definitions.length !== 1 || definition.kind !== expected) fail(`${s.name}: ${s.native.declaration} must declare exactly one ${s.native.kind}`);
      }
      return parsed.definitions;
    }),
  };
  const errors = validate(schema, document);
  if (errors.length) fail(errors.map(e => e.toString()).join("\n\n"));
  const fragments = new Map<string, FragmentDefinitionNode>();
  for (const d of document.definitions) {
    if (d.kind === Kind.FRAGMENT_DEFINITION) fragments.set(d.name.value, d);
    else if (d.kind !== Kind.OPERATION_DEFINITION) fail("Only operations and fragments are supported");
  }
  visit(document, {
    Directive(node) { fail(`Directive @${node.name.value} is not supported by this PoC`); },
  });

  function fieldsFor(type: GraphQLObjectType, sets: readonly SelectionSetNode[]): FieldPlan[] {
    const merged = new Map<string, FieldNode[]>();
    function collect(set: SelectionSetNode) {
      for (const node of set.selections) {
        if (node.kind === Kind.FIELD) {
          const key = node.alias?.value ?? node.name.value;
          if (!camelName.test(key)) fail(`Field/alias '${key}' must use lowerCamelCase`);
          const group = merged.get(key) ?? [];
          group.push(node);
          merged.set(key, group);
        } else {
          const fragment = node.kind === Kind.FRAGMENT_SPREAD ? fragments.get(node.name.value)! : node;
          if (fragment.typeCondition && fragment.typeCondition.name.value !== type.name) {
            fail("Polymorphic fragments are not supported");
          }
          collect(fragment.selectionSet);
        }
      }
    }
    sets.forEach(collect);
    return [...merged.entries()].sort(([a], [b]) => a.localeCompare(b)).map(([responseName, nodes]) => {
      const first = nodes[0];
      const name = first.name.value;
      const field = type.getFields()[name];
      if (!field) fail(`Introspection field ${name} is not supported`);
      if (field.args.some(a => a.defaultValue !== undefined)) fail(`Schema argument defaults are not supported: ${type.name}.${name}`);
      const shape = describeType(field.type);
      const args: FieldPlan["args"] = {};
      for (const arg of first.arguments ?? []) {
        const value = arg.value;
        switch (value.kind) {
          case Kind.VARIABLE: args[arg.name.value] = { variable: value.name.value }; break;
          case Kind.STRING: args[arg.name.value] = { literal: value.value }; break;
          case Kind.INT: case Kind.FLOAT: args[arg.name.value] = { literal: Number(value.value) }; break;
          case Kind.BOOLEAN: args[arg.name.value] = { literal: value.value }; break;
          case Kind.NULL: args[arg.name.value] = { literal: null }; break;
          default: fail(`Only scalar arguments are supported: ${type.name}.${name}`);
        }
      }
      const named = getNamedType(field.type);
      const children = isObjectType(named)
        ? fieldsFor(named, nodes.flatMap(n => n.selectionSet ? [n.selectionSet] : []))
        : undefined;
      if (!isObjectType(named) && !Object.hasOwn(scalars, named.name)) fail(`Unsupported output type ${named.name}`);
      return { name, responseName, coordinate: `${type.name}.${name}`, ...shape, args, ...(children ? { fields: children } : {}) };
    });
  }

  function describeType(type: GraphQLOutputType) {
    const required = isNonNullType(type);
    const bare = required ? type.ofType : type;
    const list = isListType(bare);
    if (list && (!required || !isNonNullType(bare.ofType) || isListType(bare.ofType.ofType))) {
      fail(`Only non-null lists with non-null, non-list items are supported: ${type}`);
    }
    return { type: getNamedType(type).name, required, list };
  }

  const operations: OperationPlan[] = [];
  for (const d of document.definitions) {
    if (d.kind !== Kind.OPERATION_DEFINITION) continue;
    if (d.operation !== "query" || !d.name || !/^[A-Z][a-zA-Z0-9]*$/.test(d.name.value)) {
      fail("Only named queries with PascalCase names are supported");
    }
    const variables: OperationPlan["variables"] = (d.variableDefinitions ?? []).map(v => {
      const name = v.variable.name.value;
      if (!camelName.test(name)) fail(`Variable '${name}' must use lowerCamelCase`);
      if (v.defaultValue) fail(`Variable defaults are not supported: $${name}`);
      const required = v.type.kind === Kind.NON_NULL_TYPE;
      const type = required ? v.type.type : v.type;
      if (type.kind !== Kind.NAMED_TYPE || !Object.hasOwn(scalars, type.name.value)) {
        fail(`Only scalar variables are supported: $${name}`);
      }
      return { name, type: type.name.value as Scalar, required };
    });
    operations.push({ name: d.name!.value, variables, fields: fieldsFor(schema.getQueryType()!, [d.selectionSet]) });
  }
  operations.sort((a, b) => a.name.localeCompare(b.name));
  if (!operations.length) fail("At least one named query is required");

  if (previous && previous.version !== 1) fail("Unsupported contract lock version");
  const lock: ContractLock = structuredClone(previous ?? { version: 1, messages: {} });
  const messages: string[] = [];
  function emitMessage(message: string, fields: { name: string; protoType: string; signature: string; modifier: string }[]) {
    const entry = lock.messages[message] ?? { fields: {}, retired: [] };
    const active = new Set(fields.map(f => f.name));
    for (const name of Object.keys(entry.fields)) {
      if (!active.has(name) && !entry.retired.includes(name)) entry.retired.push(name);
    }
    let next = Math.max(0, ...Object.values(entry.fields).map(f => f.number)) + 1;
    const lines = fields.map(f => {
      if (entry.retired.includes(f.name)) fail(`Retired field cannot be reused: ${message}.${f.name}`);
      let old = entry.fields[f.name];
      if (old && old.signature !== f.signature) fail(`Breaking type change: ${message}.${f.name} (${old.signature} → ${f.signature})`);
      if (!old) {
        if (next === 19000) next = 20000;
        old = entry.fields[f.name] = { number: next++, signature: f.signature };
      }
      return `  ${f.modifier}${f.protoType} ${f.name} = ${old.number};`;
    });
    entry.retired.sort();
    if (entry.retired.length) {
      lines.push(`  reserved ${entry.retired.map(n => entry.fields[n].number).sort((a,b) => a-b).join(", ")};`);
      lines.push(`  reserved ${entry.retired.map(n => JSON.stringify(n)).join(", ")};`);
    }
    lock.messages[message] = entry;
    messages.push(`message ${message} {\n${lines.join("\n")}\n}`);
  }

  function responseMessage(name: string, fields: FieldPlan[]) {
    emitMessage(name, fields.map(f => {
      const childName = `${name}_${upper(f.responseName)}`;
      if (f.fields) responseMessage(childName, f.fields);
      return {
        name: f.responseName,
        protoType: f.fields ? childName : scalars[f.type as Scalar],
        signature: `${f.type}${f.list ? "[]" : ""}${f.required ? "!" : "?"}`,
        modifier: f.list ? "repeated " : !f.fields && !f.required ? "optional " : "",
      };
    }));
  }
  for (const operation of operations) {
    emitMessage(`${operation.name}Request`, operation.variables.map(v => ({
      name: v.name, protoType: scalars[v.type], signature: `${v.type}${v.required ? "!" : "?"}`, modifier: "optional ",
    })));
    responseMessage(`${operation.name}Response`, operation.fields);
  }
  const proto = [
    '// Generated by npm run generate. Edit colocated .graphql sources instead.',
    'syntax = "proto3";', 'package app.v1;', '',
    'service AppService {',
    ...operations.map(o => `  rpc ${o.name}(${o.name}Request) returns (${o.name}Response);`),
    '}', '', ...messages,
  ].join("\n") + "\n";

  function tsShape(fields: FieldPlan[]): string {
    return `{ ${fields.map(f => {
      const scalar = f.type === "Boolean" ? "boolean" : ["Int", "Float"].includes(f.type) ? "number" : "string";
      const type = f.fields ? tsShape(f.fields) : scalar;
      const optional = !f.required || (!!f.fields && !f.list);
      return `${f.responseName}${optional ? "?" : ""}: ${type}${f.list ? "[]" : ""};`;
    }).join(" ")} }`;
  }
  const fragmentPlans = [...fragments.values()].sort((a,b) => a.name.value.localeCompare(b.name.value)).map(f => {
    const type = schema.getType(f.typeCondition.name.value);
    if (!isObjectType(type)) fail(`Only concrete object fragments are supported: ${f.name.value}`);
    const source = f.loc ? sourceOwners.get(f.loc.source) : undefined;
    return { name: f.name.value, type: type.name, fields: fieldsFor(type, [f.selectionSet]), native: source?.native };
  });
  const fragmentTypes = fragmentPlans.map(f => `export type ${f.name} = ${tsShape(f.fields)};`).join("\n") + "\n";

  return { proto, operations, lock, fragmentTypes, fragmentPlans };
}

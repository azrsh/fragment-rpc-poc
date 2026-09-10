import { Code, ConnectError } from "@connectrpc/connect";
import type { JsonObject, JsonValue } from "@bufbuild/protobuf";
import type { FieldPlan, OperationPlan } from "./plan.js";

export interface ExecutionContext {
  signal: AbortSignal;
  cache: Map<string, Promise<unknown>>;
}
type Resolver = (source: Record<string, unknown>, args: Record<string, unknown>, context: ExecutionContext) => unknown | Promise<unknown>;
export type Resolvers = Record<string, Resolver>;

function scalar(value: unknown, type: string, path: string, code: Code): string | number | boolean {
  if ((type === "String" || type === "ID") && typeof value === "string") return value;
  if (type === "ID" && typeof value === "number" && Number.isInteger(value)) return String(value);
  if (type === "Boolean" && typeof value === "boolean") return value;
  if (type === "Float" && typeof value === "number" && Number.isFinite(value)) return value;
  if (type === "Int" && typeof value === "number" && Number.isInteger(value) && value >= -2147483648 && value <= 2147483647) return value;
  throw new ConnectError(`Invalid ${type} at ${path}`, code);
}

export async function execute(plan: OperationPlan, request: Record<string, unknown>, resolvers: Resolvers, signal: AbortSignal): Promise<JsonObject> {
  const variables: Record<string, unknown> = {};
  for (const variable of plan.variables) {
    const value = request[variable.name];
    if (value == null) {
      if (variable.required) throw new ConnectError(`Missing variable $${variable.name}`, Code.InvalidArgument);
    } else variables[variable.name] = scalar(value, variable.type, `$${variable.name}`, Code.InvalidArgument);
  }
  const context: ExecutionContext = { signal, cache: new Map() };

  async function project(source: unknown, fields: FieldPlan[], path: string): Promise<JsonObject> {
    if (typeof source !== "object" || source === null || Array.isArray(source)) {
      throw new ConnectError(`Expected object at ${path}`, Code.Internal);
    }
    const record = source as Record<string, unknown>;
    const entries = await Promise.all(fields.map(async field => {
      signal.throwIfAborted();
      const fieldPath = `${path}.${field.responseName}`;
      const args = Object.fromEntries(Object.entries(field.args).map(([name, binding]) => [
        name, "variable" in binding ? variables[binding.variable] : binding.literal,
      ]));
      const resolver = Object.hasOwn(resolvers, field.coordinate) ? resolvers[field.coordinate] : undefined;
      if (!resolver && field.coordinate.startsWith("Query.")) throw new ConnectError(`Missing resolver: ${field.coordinate}`, Code.Internal);
      const value = resolver ? await resolver(record, args, context) : Object.hasOwn(record, field.name) ? record[field.name] : undefined;
      if (value == null) {
        if (field.required) throw new ConnectError(`Null for required field ${fieldPath}`, Code.Internal);
        return [field.responseName, null] as const;
      }
      const item = (value: unknown, itemPath: string): Promise<JsonObject> | string | number | boolean => {
        if (value == null) throw new ConnectError(`Null list item at ${itemPath}`, Code.Internal);
        return field.fields ? project(value, field.fields, itemPath) : scalar(value, field.type, itemPath, Code.Internal);
      };
      let output: JsonValue;
      if (field.list) {
        if (!Array.isArray(value)) throw new ConnectError(`Expected list at ${fieldPath}`, Code.Internal);
        output = await Promise.all(value.map((v,i) => item(v, `${fieldPath}[${i}]`)));
      } else output = await item(value, fieldPath);
      return [field.responseName, output] as const;
    }));
    return Object.fromEntries(entries);
  }
  return project({}, plan.fields, plan.name);
}

import type { compile } from "./compile.js";
import type { SourceFile } from "./compile.js";
import type { FieldPlan } from "../server/plan.js";
import { parse, Kind } from "graphql";

const upper = (s: string) => s[0].toUpperCase() + s.slice(1);
export interface NativeField { name: string; coordinate: string; args: FieldPlan["args"]; type: string; optional: boolean; list: boolean; object: boolean }
export interface NativeModel { name: string; schemaType: string; language: string; proto: boolean; fields: NativeField[] }
export interface NativeConversion { from: NativeModel; to: NativeModel }

export function nativeModels(output: ReturnType<typeof compile>, sources: SourceFile[]) {
  const models: NativeModel[] = [];
  const declarations = sources.filter(s => s.native).map(s => {
    const definition = parse(s.body).definitions[0];
    if (definition.kind !== Kind.OPERATION_DEFINITION && definition.kind !== Kind.FRAGMENT_DEFINITION) throw new Error("Invalid native definition");
    return { ...s.native!, name: s.native!.declaration + "Data", document: s.body, operation: definition.name!.value };
  });
  const names = new Set<string>();
  for (const declaration of declarations) {
    const key = `${declaration.language}:${declaration.name}`;
    if (names.has(key)) throw new Error(`Duplicate native generated type ${declaration.name}`);
    names.add(key);
  }
  function addModel(name: string, schemaType: string, fields: FieldPlan[], language: string, proto: boolean): void {
    const mapped = fields.map(f => {
      const type = f.fields ? `${name}_${upper(f.responseName)}` : f.type;
      if (f.fields) addModel(type, f.type, f.fields, language, proto);
      return { name: f.responseName, coordinate: f.coordinate, args: f.args, type, optional: !f.required, list: f.list, object: !!f.fields };
    });
    models.push({ name, schemaType, fields: mapped, language, proto });
  }
  for (const fragment of output.fragmentPlans) if (fragment.native) {
    addModel(fragment.native.declaration + "Data", fragment.type, fragment.fields, fragment.native.language, false);
  }
  for (const language of ["swift", "kotlin"]) {
    for (const operation of output.operations.filter(o => declarations.some(d => d.language === language && d.operation === o.name))) {
      addModel(operation.name + "Response", "Query", operation.fields, language, true);
    }
  }
  function contains(from: NativeModel, to: NativeModel): boolean {
    return from.schemaType === to.schemaType && to.fields.every(field => {
      const f = from.fields.find(f => f.name === field.name);
      if (!f || f.coordinate !== field.coordinate || JSON.stringify(f.args) !== JSON.stringify(field.args) || f.optional !== field.optional || f.list !== field.list || f.object !== field.object) return false;
      return field.object ? contains(models.find(m => m.name === f.type && m.language === from.language)!, models.find(m => m.name === field.type && m.language === to.language)!) : f.type === field.type;
    });
  }
  const conversions: NativeConversion[] = [];
  for (const to of models.filter(m => !m.proto)) for (const from of models) {
    if (from.language === to.language && from.name !== to.name && contains(from, to)) conversions.push({ from, to });
  }
  return { models, conversions, declarations };
}

const swiftScalar: Record<string, string> = { ID: "String", String: "String", Int: "Int32", Float: "Double", Boolean: "Bool" };
const protoProperty = (s: string) => s.replace(/Url$/, "URL");

export function swiftModels(native: ReturnType<typeof nativeModels>): string {
  const output = ["// Generated from Swift GraphQL declarations.\n"];
  for (const declaration of native.declarations.filter(d => d.language === "swift" && d.kind === "query")) {
    output.push(`typealias ${declaration.name} = App_V1_${declaration.operation}Response`);
  }
  for (const model of native.models.filter(m => m.language === "swift" && !m.proto)) {
    output.push(`struct ${model.name}: Equatable {\n${model.fields.map(f => {
      let type = f.object ? f.type : swiftScalar[f.type];
      if (f.list) type = `[${type}]`;
      if (f.optional) type += "?";
      return `    let ${f.name}: ${type}`;
    }).join("\n")}\n}`);
  }
  for (const { from, to } of native.conversions.filter(c => c.from.language === "swift")) {
    const name = from.proto ? `App_V1_${from.name}` : from.name;
    output.push(`extension ${name} {\n    func as${to.name}() -> ${to.name} {\n        ${to.name}(\n${to.fields.map(f => {
      const property = from.proto ? protoProperty(f.name) : f.name;
      let value = `self.${property}`;
      if (f.object) {
        if (f.list) value += `.map { $0.as${f.type}() }`;
        else value += `${f.optional && !from.proto ? "?" : ""}.as${f.type}()`;
      }
      if (f.optional && from.proto) value = `self.has${upper(property)} ? ${value} : nil`;
      return `            ${f.name}: ${value}`;
    }).join(",\n")}\n        )\n    }\n}`);
  }
  return output.join("\n\n") + "\n";
}

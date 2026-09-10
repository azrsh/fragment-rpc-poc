export type Scalar = "ID" | "String" | "Int" | "Float" | "Boolean";

export interface FieldPlan {
  name: string;
  responseName: string;
  coordinate: string;
  type: string;
  required: boolean;
  list: boolean;
  args: Record<string, { variable: string } | { literal: string | number | boolean | null }>;
  fields?: FieldPlan[];
}

export interface OperationPlan {
  name: string;
  variables: { name: string; type: Scalar; required: boolean }[];
  fields: FieldPlan[];
}

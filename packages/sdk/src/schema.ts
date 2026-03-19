/**
 * Standard Schema support for @zenv/sdk.
 *
 * Accepts any Standard Schema compliant validator (Zod, Valibot, ArkType).
 * See: https://github.com/standard-schema/standard-schema
 *
 * The schema serves two purposes:
 * 1. Fetch manifest — keys define which secrets to fetch
 * 2. Validation + transformation — values are validated after decryption
 */

/** Minimal Standard Schema v1 interface — what we actually use. */
interface StandardSchemaV1<Output = unknown> {
  readonly "~standard": {
    readonly version: 1;
    readonly vendor: string;
    readonly validate: (
      value: unknown,
    ) => StandardResult<Output> | Promise<StandardResult<Output>>;
    readonly types?: { readonly output: Output };
  };
}

type StandardResult<T> =
  | { readonly value: T; readonly issues?: undefined }
  | { readonly issues: readonly StandardIssue[] };

interface StandardIssue {
  readonly message: string;
  readonly path?: readonly (string | number | symbol)[];
}

/** Check if a value looks like a Standard Schema object schema. */
function isStandardSchema(v: unknown): v is StandardSchemaV1 {
  return (
    typeof v === "object" &&
    v !== null &&
    "~standard" in v &&
    typeof (v as any)["~standard"]?.validate === "function"
  );
}

/** Check if a value looks like a Standard Schema field (individual validator). */
function isStandardSchemaField(v: unknown): v is StandardSchemaV1 {
  return isStandardSchema(v);
}

/**
 * Extract key names from a schema definition.
 *
 * Supports:
 * - Standard Schema object (z.object({ KEY: z.string() })) — extracts from ~standard.types
 * - Plain object with Standard Schema fields ({ KEY: z.string() })
 * - Plain object with any values ({ KEY: {} }) — keys-only mode
 */
export function extractKeys(schema: Record<string, unknown>): string[] {
  // If it's a Standard Schema object itself (like z.object({...})),
  // we need to get the shape keys. Standard Schema doesn't expose shape keys
  // directly — we need to look at the types.output type or fall back to
  // checking if it has an inner shape.
  //
  // Most schema libs expose `.shape` or `._def.shape` for object schemas.
  // We check common patterns.
  const shape = getObjectShape(schema);
  if (shape) {
    return Object.keys(shape);
  }

  // Plain object — keys are the secret names
  return Object.keys(schema);
}

/** Try to extract the inner shape from a schema object (Zod, Valibot, ArkType). */
function getObjectShape(
  schema: Record<string, unknown>,
): Record<string, unknown> | null {
  // Zod: schema.shape is the object
  if ("shape" in schema && typeof schema.shape === "object" && schema.shape) {
    return schema.shape as Record<string, unknown>;
  }

  // Valibot: schema.entries
  if (
    "entries" in schema &&
    typeof schema.entries === "object" &&
    schema.entries
  ) {
    return schema.entries as Record<string, unknown>;
  }

  // ArkType: schema.infer or check if it has ~standard
  // For plain objects, just return null
  return null;
}

export interface ValidationError {
  key: string;
  message: string;
}

/**
 * Validate decrypted values against schema fields.
 *
 * Returns validated (and potentially transformed) values.
 * Collects all errors and reports them at once.
 */
export async function validateValues(
  schema: Record<string, unknown>,
  values: Record<string, string>,
): Promise<{
  result: Record<string, unknown>;
  errors: ValidationError[];
}> {
  const shape = getObjectShape(schema) ?? schema;
  const result: Record<string, unknown> = {};
  const errors: ValidationError[] = [];

  for (const [key, rawValue] of Object.entries(values)) {
    const field = shape[key];

    // No validator for this key — pass through as string
    if (!field || !isStandardSchemaField(field)) {
      result[key] = rawValue;
      continue;
    }

    // Run Standard Schema validation
    const output = await field["~standard"].validate(rawValue);

    if (output.issues) {
      const msgs = output.issues.map((i) => i.message).join("; ");
      errors.push({ key, message: msgs });
    } else {
      result[key] = output.value;
    }
  }

  return { result, errors };
}

/**
 * Infer the output type from a schema.
 * Used for generic type inference in load<T>().
 */
export type InferSchema<T> = T extends StandardSchemaV1<infer O>
  ? O
  : T extends Record<string, StandardSchemaV1<infer V>>
    ? { [K in keyof T]: V }
    : { [K in keyof T]: string };

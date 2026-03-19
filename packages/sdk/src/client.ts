/**
 * Typed HTTP client for the zEnv API.
 *
 * Generated from OpenAPI spec via openapi-fetch + openapi-typescript.
 * Types stay in sync with the Go API automatically via `make sdk-types`.
 */
import createClient from "openapi-fetch";
import type { paths } from "./api.d.ts";

export interface ClientConfig {
  baseUrl: string;
  token: string;
}

export function createApiClient(config: ClientConfig) {
  return createClient<paths>({
    baseUrl: config.baseUrl.replace(/\/+$/, ""),
    headers: {
      Authorization: `Bearer ${config.token}`,
    },
  });
}

export type ApiClient = ReturnType<typeof createApiClient>;

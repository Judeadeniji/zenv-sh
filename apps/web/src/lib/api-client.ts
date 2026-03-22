import createClient from "openapi-fetch";
import type { paths } from "./api.d.ts";
import { env } from "./env";
import { createIsomorphicFn } from "@tanstack/react-start";
import { getRequestHeaders } from "@tanstack/react-start/server";

export const api = createIsomorphicFn()
  .server(() => {
    const headers = getRequestHeaders();
    return createClient<paths>({
      baseUrl: env.VITE_API_URL,
      credentials: "include",
      headers,
    });
  })
  .client(() =>
    createClient<paths>({
      baseUrl: env.VITE_API_URL,
      credentials: "include",
    }),
  );

export type { paths };

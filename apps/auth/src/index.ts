import { serve } from "@hono/node-server";
import { Hono } from "hono";
import { cors } from "hono/cors";
import { logger } from "hono/logger";
import { auth } from "./auth.js";
import { env } from "./env.js";

const app = new Hono()
.basePath("/api");

const trustedOrigins = env.TRUSTED_ORIGINS
  ? env.TRUSTED_ORIGINS.split(",").map((s) => s.trim())
  : [];

app.use(logger());

app.use(
  "/auth/**",
  cors({
    origin: trustedOrigins,
    credentials: true,
    allowHeaders: ["Content-Type", "Authorization"],
    allowMethods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
  }),
);

app.all("/auth/**", (c) => auth.handler(c.req.raw));

app.get("/health", (c) => c.json({ status: "ok" }));

const port = env.PORT;

serve({ fetch: app.fetch, port }, (i) => {
  console.log(`zenv-auth listening on ${i.address}:${i.port}`);
});

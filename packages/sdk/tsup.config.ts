import { defineConfig } from "tsup"

export default defineConfig({
	entry: ["src/index.ts"],
	format: ["cjs", "esm"],
	dts: true,
	sourcemap: true,
	clean: true,
	splitting: false,
	treeshake: true,
	tsconfig: "tsconfig.build.json",
	// Externalize workspace deps and openapi-fetch — consumers install them.
	// Never bundle @zenv/amnesia into the SDK output.
	external: ["@zenv/amnesia", "openapi-fetch"],
})

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
	// hash-wasm and tweetnacl are runtime deps — don't bundle them.
	// Consumers install them alongside @zenv/amnesia.
	noExternal: [],
})

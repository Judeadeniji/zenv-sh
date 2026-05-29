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
	// Externalize everything that must not be bundled:
	//   vite  — peer dep, resolved by the consumer's project
	//   @zenv-sh/sdk + @zenv-sh/amnesia — workspace deps, not bundled
	external: ["vite", "@zenv-sh/sdk", "@zenv-sh/amnesia"],
})

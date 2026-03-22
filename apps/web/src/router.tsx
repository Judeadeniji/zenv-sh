import { createRouter as createTanStackRouter } from "@tanstack/react-router"
import { setupRouterSsrQueryIntegration } from "@tanstack/react-router-ssr-query"
import * as TanstackQuery from "./integrations/tanstack-query/root-provider"
import { routeTree } from "./routeTree.gen"

export function getRouter() {
	const rqContext = TanstackQuery.getContext()

	const router = createTanStackRouter({
		routeTree,
		context: { ...rqContext },
		defaultPreload: "intent",
		scrollRestoration: true,
		Wrap: (props: { children: React.ReactNode }) => {
			return <TanstackQuery.Provider {...rqContext}>{props.children}</TanstackQuery.Provider>
		},
	})

	setupRouterSsrQueryIntegration({
		router,
		queryClient: rqContext.queryClient,
	})

	return router
}

declare module "@tanstack/react-router" {
	interface Register {
		router: ReturnType<typeof getRouter>
	}
}

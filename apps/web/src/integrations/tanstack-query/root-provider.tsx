import { QueryClient, QueryClientProvider } from "@tanstack/react-query"

export function getContext() {
	const queryClient = new QueryClient({
		defaultOptions: {
			queries: {
				staleTime: 30_000,
				retry: 1,
			},
		},
	})
	return { queryClient }
}

export type MyRouteContext = ReturnType<typeof getContext>;

export function Provider({
	children,
	queryClient,
}: {
	children: React.ReactNode
	queryClient: QueryClient
}) {
	return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
}

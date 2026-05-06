import { meQueryOptions } from "#/lib/queries/auth";
import { redirect } from "@tanstack/react-router";
import { Outlet, createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_auth")({
  beforeLoad: async ({ context, location }) => {
    // Session + vault state check — works on both server and client
    // because meQueryOptions uses the isomorphic API client
    let me;

    me = await context.queryClient.fetchQuery(meQueryOptions);
    // Vault not set up → redirect to setup (skip if already there)
    if (!me.vault_setup_complete && location.pathname !== "/vault-setup") {
      throw redirect({ to: "/vault-setup" });
    }
    if (me.vault_unlocked) throw redirect({ to: "/" });
    throw redirect({ to: "/unlock" });
  },
  component: RouteComponent,
});

function RouteComponent() {
  return <Outlet />;
}

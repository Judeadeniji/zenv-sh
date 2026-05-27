package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/Judeadeniji/zenv-sh/terraform-provider-zenv/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/judeadeniji/zenv",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(), opts)
	if err != nil {
		slog.Error("provider server failed", "error", err)
		os.Exit(1)
	}
}

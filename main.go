package main

import (
	"context"
	"flag"
	"log"

	"github.com/confstack/terraform-provider-confstack/internal/adapter/driving/terraform"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/confstack/confstack",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), terraform.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}

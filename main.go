package main

import (
	"log"

	"github.com/hashicorp/terraform/plugin"
	convox "github.com/mattaitchison/terraform-provider-convox/convox"
)

var version = "0.1.6-dev"

func main() {
	log.Println("[INFO] convox provider version:", version)
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: convox.Provider,
	})
}

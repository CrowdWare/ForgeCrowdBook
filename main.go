package main

import (
	"fmt"
	"log"

	"codeberg.org/crowdware/forgecrowdbook/internal/config"
	_ "codeberg.org/crowdware/forgecrowdbook/internal/db"
)

var version = "dev"

func main() {
	cfg, err := config.LoadConfig("app.sml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	fmt.Printf("%s version %s\n", cfg.Name, version)
}

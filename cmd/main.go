package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	tggl "github.com/tggl/go-tggl-client"
)

func main() {
	apiServerKey := os.Getenv("TGGL_API_KEY")
	client := tggl.NewLocalClient(apiServerKey)

	config, err := client.GetConfig()
	if err != nil {
		log.Fatalf("Error getting config: %v", err)
	}

	prettyJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Fatalf("Error formatting JSON: %v", err)
	}

	fmt.Printf("Configuration:\n%s\n", string(prettyJSON))
}

package main

import (
	"log"
	"net/http"
	"os"

	"github.com/tggl/go-tggl-client"
)

func main() {
	apiServerKey := os.Getenv("TGGL_API_KEY")
	client := tggl.NewLocalClient(apiServerKey, &http.Client{}, tggl.WithPollingInterval(3000))

	if err := client.GetConfig(); err != nil {
		log.Fatalf("Error getting config: %v", err)
	}

	/*	client.Get(tggl.Context{
		"abc": "1",
	}, "mars", "avril")*/
}

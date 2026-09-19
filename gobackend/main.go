// Command gobackend serves the AI UML Architect public API skeleton (GOBE-01).
package main

import (
	"log"
	"net/http"

	"github.com/ai-uml-architect/gobackend/internal/httpapi"
)

func main() {
	log.Fatal(http.ListenAndServe(":8081", httpapi.NewMux()))
}

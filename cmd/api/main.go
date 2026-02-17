package main

import (
	"log"
	"net/http"

	"github.com/n1ckerr0r/shortener/core/application/create_link"
	"github.com/n1ckerr0r/shortener/core/application/resolve_link"
	"github.com/n1ckerr0r/shortener/infrastructure/clock"
	"github.com/n1ckerr0r/shortener/infrastructure/generator"
	"github.com/n1ckerr0r/shortener/infrastructure/repository"

	httpin "github.com/n1ckerr0r/shortener/adapters/inbound/http"
)

func main() {
	repo := repository.NewMemoryRepository()
	clk := clock.SystemClock{}
	gen := generator.SimpleGenerator{}

	createService := create_link.NewService(repo, gen, clk)
	resolveService := resolve_link.NewService(repo, clk)

	createHandler := httpin.NewCreateLinkHandler(createService)
	redirectHandler := httpin.NewRedirectHandler(resolveService)

	mux := http.NewServeMux()

	mux.Handle("/links", createHandler)
	mux.Handle("/", redirectHandler)

	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

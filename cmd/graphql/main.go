package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	gographql "github.com/graphql-go/graphql"
	"github.com/inconshreveable/log15"
	"github.com/joho/godotenv"
	"neodeliver.com/engine/db"
	"neodeliver.com/engine/graphql"
	"neodeliver.com/modules/campaigns"
	"neodeliver.com/modules/contacts"
	"neodeliver.com/modules/settings"
)

func main() {
	log15.Root().SetHandler(log15.LvlFilterHandler(log15.LvlDebug, log15.StreamHandler(os.Stderr, log15.TerminalFormat())))
	log15.Info("Starting graphql server...")

	godotenv.Overload()
	defer db.Close()

	// create schema
	scheme := graphql.New()
	settings.Init(scheme)
	contacts.Init(scheme)
	campaigns.Init(scheme)

	instance, err := scheme.Build()
	if err != nil {
		panic(err)
	}

	if os.Getenv("TEST") == "1" {
		testSchema(instance)
	} else {
		httpServer(instance)
	}
}

func httpServer(s gographql.Schema) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log15.Info("Listening http server", "port", port)
	http.HandleFunc("/", graphql.Route(s))
	http.ListenAndServe(":"+port, nil)
}

func testSchema(s gographql.Schema) {
	query := `
		{
			users {
				id
				name
				email
			}
		}
	`

	params := gographql.Params{Schema: s, RequestString: query, Context: context.Background()}
	r := gographql.Do(params)
	if len(r.Errors) > 0 {
		log15.Error("failed to execute graphql operation, errors", "err", r.Errors)
		panic("failed to execute graphql operation")
	}

	rJSON, _ := json.Marshal(r)
	fmt.Printf("%s \n", rJSON)
}

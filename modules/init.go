package modules

import (
	gographql "github.com/graphql-go/graphql"
	"github.com/inconshreveable/log15"
	"github.com/joho/godotenv"
	"neodeliver.com/engine/db"
	"neodeliver.com/engine/graphql"
	"neodeliver.com/modules/campaigns"
	"neodeliver.com/modules/contacts"
	"neodeliver.com/modules/settings"
)

func Build() gographql.Schema {
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

	return instance
}

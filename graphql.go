// Package helloworld provides a set of Cloud Functions samples.
package graphql

import (
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"neodeliver.com/engine/graphql"
	"neodeliver.com/modules"
)

func init() {
	instance := modules.Build()
	functions.HTTP("GraphQL", graphql.Route(instance))
}

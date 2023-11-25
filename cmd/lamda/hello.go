package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"neodeliver.com/engine/graphql"
	"neodeliver.com/modules"
)

func main() {
	instance := modules.Build()
	http.HandleFunc("/", graphql.Route(instance))

	if runtime_api, _ := os.LookupEnv("AWS_LAMBDA_RUNTIME_API"); runtime_api != "" {
		fmt.Println("Starting up in Lambda Runtime")
		adapter := httpadapter.New(http.DefaultServeMux).ProxyWithContext
		lambda.Start(adapter)
	} else {
		port := os.Getenv("PORT")
		fmt.Println("Starting up http server on port " + port)
		srv := &http.Server{
			Addr: ":" + port,
		}

		err := srv.ListenAndServe()
		if err != nil {
			fmt.Println("Could not start server: ", err)
		}
	}
}

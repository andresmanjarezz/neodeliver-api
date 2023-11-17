package graphql

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/graphql-go/graphql"
	"neodeliver.com/engine/rbac"
)

func Route(schema graphql.Schema) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload := struct {
			Query         string                 `json:"query"`
			Variables     map[string]interface{} `json:"variables"`
			OperationName string                 `json:"operationName"`
		}{}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && err != io.EOF {
			http.Error(w, err.Error(), 400)
		} else if payload.Query == "" && r.Method == "GET" {
			bs := ServePlayground()
			w.Header().Set("Content-Type", "text/html")
			w.Write(bs)
			return
		} else if payload.Query == "" {
			http.Error(w, "no query provided", 400)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, "rbac", func() (rbac.RBAC, error) {
			return rbac.Load(r)
		})

		ctx = context.WithValue(ctx, "client_ip", "::1") // TODO set client ip

		result := graphql.Do(graphql.Params{
			Schema:         schema,
			RequestString:  payload.Query,
			VariableValues: payload.Variables,
			OperationName:  payload.OperationName,
			Context:        ctx,
		})

		json.NewEncoder(w).Encode(result)
	}
}

func ServePlayground() []byte {
	return []byte(`<!DOCTYPE html>
	<html>
	
	<head>
	  <meta charset=utf-8/>
	  <meta name="viewport" content="user-scalable=no, initial-scale=1.0, minimum-scale=1.0, maximum-scale=1.0, minimal-ui">
	  <title>GraphQL Playground</title>
	  <link rel="stylesheet" href="//cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css" />
	  <link rel="shortcut icon" href="//cdn.jsdelivr.net/npm/graphql-playground-react/build/favicon.png" />
	  <script src="//cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
	</head>
	
	<body>
	  <div id="root">
		<style>
		  body {
			background-color: rgb(23, 42, 58);
			font-family: Open Sans, sans-serif;
			height: 90vh;
		  }
	
		  #root {
			height: 100%;
			width: 100%;
			display: flex;
			align-items: center;
			justify-content: center;
		  }
	
		  .loading {
			font-size: 32px;
			font-weight: 200;
			color: rgba(255, 255, 255, .6);
			margin-left: 20px;
		  }
	
		  img {
			width: 78px;
			height: 78px;
		  }
	
		  .title {
			font-weight: 400;
		  }
		</style>
		<img src='//cdn.jsdelivr.net/npm/graphql-playground-react/build/logo.png' alt=''>
		<div class="loading"> Loading
		  <span class="title">GraphQL Playground</span>
		</div>
	  </div>
	  <script>window.addEventListener('load', function (event) {
		  GraphQLPlayground.init(document.getElementById('root'), {
			// options as 'endpoint' belong here
		  })
		})</script>
	</body>
	
	</html>`)
}

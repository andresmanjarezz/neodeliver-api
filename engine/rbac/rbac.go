package rbac

import (
	"context"
	"net/http"
	"os"
)

type RBAC struct {
	UserID         string
	OrganizationID string
	Token          string
	Scopes         map[string]bool
}

func Load(req *http.Request) (RBAC, error) {
	sub := os.Getenv("AUTH0_SUB")
	if sub == "" {
		sub = "auth0|655c75b291c2f4235db683fa"
	}

	// TODO load rbac from request
	return RBAC{
		UserID:         sub,
		OrganizationID: "56cde8c6-5af5-11ee-8c99-0242ac120002",
		// TODO : The token env is only for testing, it should be replaced with
		// the actual access token of the user
		Token: os.Getenv("AUTH0_ACCESS_TOKEN"),
		Scopes: map[string]bool{
			"users:read": true,
		},
	}, nil
}

func FromContext(ctx context.Context) (RBAC, error) {
	if rbac, ok := ctx.Value("rbac").(func() (RBAC, error)); ok {
		return rbac()
	}

	return RBAC{}, nil
}

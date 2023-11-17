package rbac

import (
	"context"
	"net/http"
)

type RBAC struct {
	SUB            string
	UserID         string
	OrganizationID string
	Scopes         map[string]bool
}

func Load(req *http.Request) (RBAC, error) {
	// TODO load rbac from request
	return RBAC{
		SUB:            "oauth0|token",
		UserID:         "328f9c39-9fff-40e1-bccd-40a0d7506fa7",
		OrganizationID: "56cde8c6-5af5-11ee-8c99-0242ac120002",
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

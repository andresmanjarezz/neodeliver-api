package settings

import (
	"errors"
	"fmt"

	"github.com/graphql-go/graphql"
	"neodeliver.com/engine/rbac"
)

// queries:
type SecuritySettings struct {
	TwoFactorEnabled bool `bson:"two_factor_enabled" json:"two_factor_enabled"`
	// ...
}

func (Query) SecuritySettings(p graphql.ResolveParams, rbac rbac.RBAC) (SecuritySettings, error) {
	// TODO
	return SecuritySettings{}, nil
}

// mutations:

type UpdatePassword struct {
	Old string `bson:",omitempty" validate:"omitempty,lte=255"`
	New string `bson:",omitempty" validate:"omitempty,lte=255"`
}

// UpdatePassword updates the password at Auth0.
func (Mutation) UpdatePassword(p graphql.ResolveParams, rbac rbac.RBAC, args UpdatePassword) (bool, error) {
	// TODO verify if current password is correct

	auth := Auth0()
	res := struct {
		StatusCode int    `json:"statusCode"`
		Message    string `json:"message"`
		Error      string
	}{}

	url := fmt.Sprintf("/api/v2/users/%s", rbac.UserID)
	bs, status, err := auth.Patch(p.Context, url, map[string]interface{}{
		"password": args.New,
	}, &res)

	// verify response
	if err != nil {
		fmt.Println(err)
		// TODO log to sentry
		return false, errors.New("internal_error")
	} else if res.StatusCode == 400 {
		return false, fmt.Errorf(res.Message)
	} else if status != 200 {
		fmt.Println(string(bs))
		return false, fmt.Errorf("failed to update password")
	}

	return true, err
}

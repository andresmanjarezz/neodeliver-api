package settings

import (
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

	// get management access token
	token, err := getManagementToken()
	if err != nil {
		return false, fmt.Errorf("failed to get Auth0 management token: %v", err)
	}

	// update user password
	err = updateUserPassword(token, rbac.UserID, args)
	if err != nil {
		return false, err
	}

	return true, nil
}

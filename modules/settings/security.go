package settings

import (
	"errors"

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

func (Mutation) UpdatePassword(p graphql.ResolveParams, rbac rbac.RBAC, args EditUser) (bool, error) {
	// TODO
	return false, errors.New("not implemented yet")
}

package settings

import (
	"errors"

	"github.com/graphql-go/graphql"
	"neodeliver.com/engine/rbac"
)

type UpdatePassword struct {
	Old string `bson:",omitempty" validate:"omitempty,lte=255"`
	New string `bson:",omitempty" validate:"omitempty,lte=255"`
}

func (Mutation) UpdatePassword(p graphql.ResolveParams, rbac rbac.RBAC, args EditUser) (bool, error) {
	return false, errors.New("not implemented yet")
}

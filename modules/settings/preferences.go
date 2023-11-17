package settings

import (
	"github.com/graphql-go/graphql"
	"neodeliver.com/engine/db"
	"neodeliver.com/engine/rbac"
)

type User struct {
	ID             string `bson:"_id"`
	OrganizationID string `bson:"organization_id"`
	Name           string
	Title          string
	Lang           string
	Notifications  UserNotifications
}

type UserNotifications struct {
	Tips       bool
	Updates    bool
	Promotions bool
	Security   bool
	Reports    bool
}

// ----------------------------------------
// edit user

type EditUser struct {
	Name          *string            `bson:",omitempty" validate:"omitempty,lte=150"`
	Title         *string            `bson:",omitempty" validate:"omitempty,lte=150"`
	Lang          *string            `bson:",omitempty" validate:"omitempty,oneof=en de fr nl"`
	Notifications *UserNotifications `bson:",omitempty"`
}

func (Mutation) EditUser(p graphql.ResolveParams, rbac rbac.RBAC, args EditUser) (User, error) {
	u := User{}
	err := db.Update(p.Context, &u, map[string]interface{}{
		"_id": rbac.UserID,
	}, args)

	return u, err
}

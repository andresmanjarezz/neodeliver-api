package settings

import (
	"time"

	"github.com/graphql-go/graphql"
	"neodeliver.com/engine/db"
	"neodeliver.com/engine/rbac"
)

// A user is a person that connects to the interface, users can be available within multiple teams
// The data available within the user object is also visible from other organization managers
type User struct {
	ID             string `bson:"_id"`
	Name           string
	Email          string
	Title          string
	Lang           string
	TimeZone       string
	TimeFormat     string
	Country        string
	ProfilePicture string // TODO allow to upload pictures
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ----------------------------------------
// edit user

type EditUser struct {
	Name  *string `bson:",omitempty" validate:"omitempty,lte=150"`
	Title *string `bson:",omitempty" validate:"omitempty,lte=150"`
	Lang  *string `bson:",omitempty" validate:"omitempty,oneof=en de fr nl"`
}

func (Mutation) EditUser(p graphql.ResolveParams, rbac rbac.RBAC, args EditUser) (User, error) {
	// TODO add & validate input types:
	// Email      string
	// Lang       string
	// TimeZone   string
	// TimeFormat string
	// Country    string

	u := User{}
	err := db.Update(p.Context, &u, map[string]interface{}{
		"_id": rbac.UserID,
	}, args)

	// TODO update data from team members to keep it in sync (only if index fields have been changed)
	// TODO update data from our own organization contacts list

	return u, err
}

package settings

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/graphql-go/graphql"
	"neodeliver.com/engine/db"
	"neodeliver.com/engine/rbac"
)

// A user is a person that connects to the interface, users can be available within multiple teams
// The data available within the user object is also visible from other organization managers
type User struct {
	ID            string     `json:"user_id" graphql:"id"`
	Name          string     `json:"name"`
	Email         string     `json:"email"`
	EmailVerified bool       `json:"email_verified"`
	Identities    []Identity `json:"identities"`
	Picture       string     `json:"picture"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	Error            string `json:",omitempty" graphql:"-"`
	ErrorDescription string `json:"error_description,omitempty" graphql:"-"`

	UserMetadata `json:"user_metadata" graphql:",inline"`
}

type Identity struct {
	UserID     string `json:"user_id"`
	Provider   string `json:"provider"`
	Connection string `json:"connection"`
	Social     bool   `json:"isSocial"`
}

type UserMetadata struct {
	Title      string `json:"title"`
	Lang       string `json:"lang"`
	TimeZone   string `json:"time_zone"`
	TimeFormat string `json:"time_format"`
	Country    string `json:"country"`
}

func GetUserByID(ctx context.Context, id string) (User, error) {
	auth := Auth0()
	var res User

	url := fmt.Sprintf("/api/v2/users/%s", id)
	_, _, err := auth.Get(ctx, url, nil, &res)
	if err != nil {
		return res, errors.New("internal error")
	} else if res.ErrorDescription != "" {
		return res, errors.New(res.ErrorDescription)
	}

	return res, nil
}

// fetches user information from auth0
func (Query) User(p graphql.ResolveParams, rbac rbac.RBAC) (User, error) {
	return GetUserByID(p.Context, rbac.UserID)
}

// ----------------------------------------
// edit user

type EditUser struct {
	Name       *string // `validate:"lte=150"`
	Title      *string // `validate:"lte=150"`
	Lang       *string // `validate:"oneof=en de fr nl"`
	TimeZone   *string `json:"time_zone"`
	TimeFormat *string `json:"time_format"`
	Country    *string `json:"country"`
	Email      *string `json:"email"`
	// Picture       string     `json:"picture"`
}

func (Mutation) EditUser(p graphql.ResolveParams, rbac rbac.RBAC, args EditUser) (User, error) {
	user, up, err := updateAuth0User(p, rbac, args)

	// update indexed team data
	if mapContainsKey(up, "name", "email", "picture") {
		err = db.Update(p.Context, &TeamMember{}, map[string]interface{}{
			"user_id": user.ID,
		}, map[string]interface{}{
			"name":            user.Name,
			"email":           user.Email,
			"profile_picture": user.Picture,
			"updated_at":      time.Now(),
		})

		if err != nil {
			return user, err
		}
	}

	// TODO update data from our own organization contacts list

	return user, err
}

// update user data on auth0 & return updated data & fields
func updateAuth0User(p graphql.ResolveParams, rbac rbac.RBAC, args EditUser) (User, map[string]interface{}, error) {
	// TODO validate input types

	// get current user info to apply updates on
	user, err := GetUserByID(p.Context, rbac.UserID)
	if err != nil {
		return user, nil, err
	}

	// find out what fields have been changed
	updates := map[string]interface{}{}
	if args.Name != nil && *args.Name != user.Name {
		updates["name"] = *args.Name
		user.Name = *args.Name
	}

	if args.Email != nil && *args.Email != user.Email {
		updates["email"] = *args.Email
		updates["verify_email"] = true
		user.Email = *args.Email
	}

	addMeta := func(dest *string, value *string) {
		if value != nil && *value != *dest {
			*dest = *value
			updates["user_metadata"] = user.UserMetadata
		}
	}

	addMeta(&user.Title, args.Title)
	addMeta(&user.Lang, args.Lang)
	addMeta(&user.TimeZone, args.TimeZone)
	addMeta(&user.TimeFormat, args.TimeFormat)
	addMeta(&user.Country, args.Country)

	// nothing to update
	if len(updates) == 0 {
		return user, updates, nil
	}

	// send updates to auth0
	url := fmt.Sprintf("/api/v2/users/%s", rbac.UserID)

	_, _, err = auth.Patch(p.Context, url, nil, updates, &user)
	if err != nil {
		return user, updates, errors.New("internal error")
	} else if user.ErrorDescription != "" {
		return user, updates, errors.New(user.ErrorDescription)
	} else if user.Error != "" {
		return user, updates, errors.New(user.Error)
	}

	return user, updates, err
}

func mapContainsKey(m map[string]interface{}, key ...string) bool {
	for _, k := range key {
		if _, ok := m[k]; ok {
			return true
		}
	}

	return false
}

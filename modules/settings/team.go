package settings

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/graphql-go/graphql"
	"github.com/segmentio/ksuid"
	"neodeliver.com/engine/db"
	"neodeliver.com/engine/rbac"
)

const TeamMemberPrefixID = "tmbr_"

type TeamMember struct {
	ID                  string     `bson:"_id"`
	OrganizationID      string     `json:"organization_id" bson:"organization_id"`
	UserID              string     `json:"user_id" bson:"user_id"`
	Role                string     `json:"role" bson:"role"`
	Name                string     `json:"name" bson:"name"`
	Email               string     `json:"email" bson:"email"`
	ProfilePicture      string     `json:"profile_picture" bson:"profile_picture"`
	InvitationExpiresAt *time.Time `json:"invitation_expires_at" bson:"invitation_expires_at"`
	CreatedAt           time.Time  `json:"created_at" bson:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" bson:"updated_at"`
	DeletedAt           *time.Time `graphql:"-" bson:"deleted_at"`
}

// ---

type InviteArgs struct {
	Email string `validate:"required,email"`
	Role  string `bson:",omitempty" validate:"omitempty,oneof=admin billing"` // TODO add more roles
}

func (Mutation) InviteUser(p graphql.ResolveParams, rbac rbac.RBAC, args InviteArgs) (TeamMember, error) {
	// verify if member already exists in the team
	u := TeamMember{}
	err := db.Find(p.Context, &u, map[string]interface{}{
		"deleted_at":      nil,
		"organization_id": rbac.OrganizationID,
		"email":           args.Email,
	})

	if err != nil && err.Error() != "mongo: no documents in result" {
		return u, err
	} else if err == nil && u.InvitationExpiresAt == nil {
		if u.Role != args.Role {
			return u, errors.New("user already exists in team with different role")
		}

		// user already accepted invitation, nothing to do
		return u, nil
	}

	// add user to team if not yet found
	if err != nil {
		exp := time.Now().Add(time.Hour * 24 * 7)
		u = TeamMember{
			ID:                  TeamMemberPrefixID + ksuid.New().String(),
			OrganizationID:      rbac.OrganizationID,
			Email:               args.Email,
			InvitationExpiresAt: &exp,
			Role:                args.Role,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}

		if err = db.Save(p.Context, &u); err != nil {
			return u, err
		}
	} else if u.Role != args.Role {
		// update member role if new invited role is different and invitation is not yet accepted
		err = db.Update(p.Context, &u, map[string]interface{}{
			"_id": u.ID,
		}, map[string]interface{}{
			"role": args.Role,
		})

		if err != nil {
			return u, err
		}
	}

	// generate invitation token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, invitationClaims{
		ID:  u.ID,
		Exp: u.InvitationExpiresAt.Unix(),
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(invitationsSecret())
	if err != nil {
		fmt.Println(err) // TODO log to sentry
		return u, nil
	}

	// send invitation email to user
	fmt.Println("todo send invitation email with token: ", tokenString)
	// TODO send invitation mail containing acceptation token (using our internal systems)

	return u, nil
}

// ---

type AcceptInviteArgs struct {
	Token string
}

// accept invitation to join team
func (Mutation) AcceptInvitation(p graphql.ResolveParams, rbac rbac.RBAC, args AcceptInviteArgs) (TeamMember, error) {
	// parse jwt token
	claims := invitationClaims{}
	_, err := jwt.ParseWithClaims(args.Token, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return invitationsSecret(), nil
	})

	if err != nil {
		return TeamMember{}, err
	}

	// find team member
	u := TeamMember{}
	err = db.Find(p.Context, &u, map[string]interface{}{
		"_id":        claims.ID,
		"deleted_at": nil,
	})

	if err != nil && err.Error() != "mongo: no documents in result" {
		return u, err
	} else if err != nil {
		return u, errors.New("invitation not found or alraedy accepted")
	}

	// verify if invitation is still valid
	if u.UserID != "" && u.UserID == rbac.UserID {
		// user already accepted invitation, nothing to do
		return u, nil
	} else if u.UserID != "" {
		return u, errors.New("invitation already accepted by another user")
	} else if u.InvitationExpiresAt != nil && time.Now().After(*u.InvitationExpiresAt) {
		return u, errors.New("invitation expired")
	}

	// update team member
	user := User{}
	err = db.Find(p.Context, &user, map[string]interface{}{
		"_id": rbac.UserID,
	})

	if err != nil {
		return u, err
	}

	err = db.Update(p.Context, &u, map[string]interface{}{
		"_id": claims.ID,
	}, map[string]interface{}{
		"user_id":               rbac.UserID,
		"name":                  user.Name,
		"email":                 user.Email,
		"profile_picture":       user.ProfilePicture,
		"invitation_expires_at": nil,
		"updated_at":            time.Now(),
	})

	return u, err
}

// --------------------------------------------
// invitation JWT helper functions

type invitationClaims struct {
	ID  string `json:"id"`
	Exp int64  `json:"exp"`
}

func (i invitationClaims) Valid() error {
	if time.Unix(i.Exp, 0).After(time.Now()) {
		return nil
	}

	return fmt.Errorf("invitation token expired")
}

func invitationsSecret() []byte {
	if s := os.Getenv("INVITATIONS_SECRET"); s != "" {
		return []byte(s)
	}

	return []byte("dev_token")
}

package settings

import "time"

type TeamMember struct {
	ID             string `bson:"_id"`
	OrganizationID string
	UserID         string
	Role           string
	Name           string
	Email          string
	ProfilePicture string
	CreatedAt      time.Time
	DeletedAt      *time.Time `graphql:"-"`
}

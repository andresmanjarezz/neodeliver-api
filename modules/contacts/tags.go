package contacts

import "time"

type Tag struct {
	ID             string `bson:"_id"`
	OrganizationID string `bson:"organization_id"`
	Name           string
	ContactsCount  int       `json:"contacts_count"`
	CreatedAt      time.Time `json:"created_at"`
}

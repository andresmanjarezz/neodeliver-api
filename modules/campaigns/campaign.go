package campaigns

import "time"

type Campaign struct {
	ID             string `bson:"_id"`
	OrganizationID string `json:"organization_id"`
	Transactional  bool
	Draft          bool
	CreatedAt      time.Time `json:"created_at"`
}

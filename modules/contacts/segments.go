package contacts

type Segment struct {
	ID             string `bson:"_id"`
	OrganizationID string `json:"organization_id"`
	Name           string
	// TODO
}

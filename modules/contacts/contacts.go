package contacts

import (
	"time"

	"github.com/graphql-go/graphql"
	"github.com/segmentio/ksuid"
	"neodeliver.com/engine/db"
	"neodeliver.com/engine/rbac"
)

type ContactStats struct {
	SMS           ContactStatsItem
	Email         ContactStatsItem
	Notifications ContactStatsItem
}

type ContactStatsItem struct {
	CampaignsSent      int
	LastCampaignSent   time.Time
	MessagesOpened     int
	LastMessageOpened  time.Time
	MessagesClicked    int
	LastMessageClicked time.Time
}

// ----

type ContactData struct {
	ExternalID         *string  `bson:"external_id" json:"external_id"` // used to map to external systems => unique per org
	GivenName          *string  `bson:"given_name" json:"given_name"`
	LastName           *string  `bson:"last_name" json:"last_name"`
	Email              *string  `bson:"email" json:"email"`
	NotificationTokens []string `bson:"notification_tokens" json:"notification_tokens"`
	PhoneNumber        *string  `bson:"phone_number" json:"phone_number"`
	Lang               string   `bson:"lang" json:"lang"`
}

func (c ContactData) Validate() error {
	// TODO verify email & phone format
	// TODO very language known
	// TODO verify notification tokens format

	return nil
}

type Contact struct {
	ID             string    `json:"id" bson:"_id,omitempty"`
	OrganizationID string    `bson:"organization_id"`
	Status         string    `bson:"status" json:"status"`
	SubscribedAt   time.Time `bson:"subscribed_at" json:"subscribed_at"`
	ContactData    `bson:",inline" json:",inline"`
}

type ContactID struct {
	ID string `bson:"_id, omitempty"`
}

func (Mutation) AddContact(p graphql.ResolveParams, rbac rbac.RBAC, args ContactData) (Contact, error) {
	c := Contact{
		ID:             "ctc_" + ksuid.New().String(),
		OrganizationID: rbac.OrganizationID,
		Status:         "ACTIVE", // Assuming a default status
		SubscribedAt:   time.Now(),
		ContactData:    args,
	}

	if err := c.Validate(); err != nil {
		return c, err
	}

	// TODO assert unique email within org
	// TODO verify external id is unique within org or override data

	_, err := db.Save(p.Context, &c)
	return c, err
}

// ---

type ContactEdit struct {
	ID   string
	Data ContactData // TODO support inline
}

func (Mutation) UpdateContact(p graphql.ResolveParams, rbac rbac.RBAC, args ContactEdit) (Contact, error) {
	if err := args.Data.Validate(); err != nil {
		return Contact{}, err
	}

	// TODO assert unique email if changed
	// TODO verify external id is unique within org or override data

	// Save the updated contact to the database
	c := Contact{}
	err := db.Update(p.Context, &c, map[string]string{
		"_id": args.ID,
	}, args.Data)

	return c, err
}

func (Mutation) DeleteContact(p graphql.ResolveParams, rbac rbac.RBAC, filter ContactID) (bool, error) {
	c := Contact{}
	err := db.Delete(p.Context, &c, map[string]string{"_id": filter.ID})
	return true, err
}

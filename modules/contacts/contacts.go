package contacts

import (
	"time"

	"github.com/graphql-go/graphql"
	"neodeliver.com/engine/rbac"
	"neodeliver.com/engine/db"
)

type Contact struct {
	ID                 string `bson:"_id, omitempty"`
	OrganizationID     string `bson:"organization_id"`
	GivenName          string `bson:"given_name"`
	LastName           string `bson:"last_name"`
	Email              string `bson:"email"`
	NotificationTokens []string `bson:"notification_tokens"`
	PhoneNumber        string `bson:"phone_number"`
	Status             string `bson:"status"`
	SubscribedAt       time.Time `bson:"subscribed_at"`
	Lang               string `bson:"lang"`
	// Meta               map[string]interface{}
	// Tags 						 []string
	// SubscribedChannels []string
	// Stats ContactStats
}

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

func (Mutation) AddContact(p graphql.ResolveParams, rbac rbac.RBAC, args Contact) (*Contact, error) {
	err := db.Save(p.Context, &args)
	return &args, err
}

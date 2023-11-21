package contacts

import (
	"time"

	"github.com/graphql-go/graphql"
	"neodeliver.com/engine/rbac"
	"neodeliver.com/engine/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Contact struct {
	ID				   string `json:"id" bson:"_id,omitempty"`
	OrganizationID     string `bson:"organization_id" json:"organization_id"`
	GivenName          string `bson:"given_name" json:"given_name"`
	LastName           string `bson:"last_name" json:"last_name"`
	Email              string `bson:"email" json:"email"`
	NotificationTokens []string `bson:"notification_tokens" json:"notification_tokens"`
	PhoneNumber        string `bson:"phone_number" json:"phone_number"`
	Status             string `bson:"status" json:"status"`
	SubscribedAt       time.Time `bson:"subscribed_at" json:"subscribed_at"`
	Lang               string `bson:"lang" json:"lang"`
}

type AddContact struct {
	OrganizationID     string `bson:"organization_id" json:"organization_id"`
	GivenName          string `bson:"given_name" json:"given_name"`
	LastName           string `bson:"last_name" json:"last_name"`
	Email              string `bson:"email" json:"email"`
	PhoneNumber        string `bson:"phone_number" json:"phone_number"`
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

type ContactID struct {
	ID                 string `bson:"_id, omitempty"`
}

func (Mutation) AddContact(p graphql.ResolveParams, rbac rbac.RBAC, args AddContact) (*Contact, error) {
	c := Contact{
		LastName:        args.LastName,
		Email:           args.Email,
		PhoneNumber:     args.PhoneNumber,
		OrganizationID:  args.OrganizationID,
		GivenName:       args.GivenName,
		Status:          "ACTIVE", // Assuming a default status
		SubscribedAt:    time.Now(), // Setting the current time as the subscribed_at value
		Lang: 			 "english",
		NotificationTokens: make([]string, 0),
	}

	insertResult, err := db.Save(p.Context, &c)
	if err != nil {
		return nil, err
	}
	filter := bson.M{"_id": insertResult.InsertedID}
	_, err = db.Find(p.Context, &c, filter)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (Mutation) UpdateContact(p graphql.ResolveParams, rbac rbac.RBAC, args Contact) (*Contact, error) {
	c := Contact{
		LastName:        args.LastName,
		Email:           args.Email,
		PhoneNumber:     args.PhoneNumber,
		OrganizationID:  args.OrganizationID,
		GivenName:       args.GivenName,
		Status:          args.Status,
		SubscribedAt:    args.SubscribedAt,
		Lang: 			 args.Lang,
		NotificationTokens: args.NotificationTokens,
	}
	objectID, _ := primitive.ObjectIDFromHex(args.ID)
	filter := bson.M{"_id": objectID}

	// Save the updated contact to the database
	err := db.Update(p.Context, &c, filter, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (Mutation) DeleteContact(p graphql.ResolveParams, rbac rbac.RBAC, filter ContactID) (bool, error) {
	c := Contact{}
	objectID, _ := primitive.ObjectIDFromHex(filter.ID)
	err := db.Delete(p.Context, &c, bson.M{"_id": objectID})
	return false, err
}

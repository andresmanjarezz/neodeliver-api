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
	OrganizationID     string `bson:"organization_id"`
	GivenName          string `bson:"given_name"`
	LastName           string `bson:"last_name"`
	Email              string `bson:"email"`
	NotificationTokens []string `bson:"notification_tokens"`
	PhoneNumber        string `bson:"phone_number"`
	Status             string `bson:"status"`
	SubscribedAt       time.Time `bson:"subscribed_at"`
	Lang               string `bson:"lang"`
}

type AddContact struct {
	OrganizationID     string `bson:"organization_id"`
	GivenName          string `bson:"given_name"`
	LastName           string `bson:"last_name"`
	Email              string `bson:"email"`
	PhoneNumber        string `bson:"phone_number"`
}

type EditContact struct {
	ID 				string `json:"id"`
	LastName        string `json:"last_name"`
	Email           string `json:"email"`
	PhoneNumber     string `json:"phone_number"`
	OrganizationID  string `json:"organization_id"`
	GivenName       string `json:"given_name"`
	Status          string `bson:"status"`
	SubscribedAt    time.Time `bson:"subscribed_at"`
	Lang            string `bson:"lang"`
	NotificationTokens []string `bson:"notification_tokens"`
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

func (Mutation) UpdateContact(p graphql.ResolveParams, rbac rbac.RBAC, args EditContact) (*Contact, error) {
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

	d := Contact{}
	_, err := db.Find(p.Context, &d, filter)
	if err != nil {
		return nil, err
	}

	// Save the updated contact to the database
	err = db.Update(p.Context, &c, filter, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (Mutation) DeleteContact(p graphql.ResolveParams, rbac rbac.RBAC, filter ContactID) (bool, error) {
	c := Contact{}
	objectID, _ := primitive.ObjectIDFromHex(filter.ID)
	err := db.Delete(p.Context, &c, bson.M{"_id": objectID})
	if err != nil {
		return false, err
	}
	return true, nil
}

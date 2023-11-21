package contacts

import (
	"time"
	"fmt"

	"github.com/graphql-go/graphql"
	"neodeliver.com/engine/rbac"
	"neodeliver.com/engine/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

type AddContact struct {
	OrganizationID     string `bson:"organization_id"`
	GivenName          string `bson:"given_name"`
	LastName           string `bson:"last_name"`
	Email              string `bson:"email"`
	PhoneNumber        string `bson:"phone_number"`
	// Meta               map[string]interface{}
	// Tags 						 []string
	// SubscribedChannels []string
	// Stats ContactStats
}

type AddContactResponse struct {
	ID                 string `bson:"_id, omitempty"`
	OrganizationID     string `bson:"organization_id"`
	GivenName          string `bson:"given_name"`
	LastName           string `bson:"last_name"`
	Email              string `bson:"email"`
	PhoneNumber        string `bson:"phone_number"`
	// Meta               map[string]interface{}
	// Tags 						 []string
	// SubscribedChannels []string
	// Stats ContactStats
}

type EditContact struct {
	ID                 string `bson:"_id, omitempty"`
	OrganizationID     string `bson:"organization_id"`
	GivenName          string `bson:"given_name"`
	LastName           string `bson:"last_name"`
	Email              string `bson:"email"`
	PhoneNumber        string `bson:"phone_number"`
	// Meta               map[string]interface{}
	// Tags 						 []string
	// SubscribedChannels []string
	// Stats ContactStats
}

type ContactID struct {
	ID                 string `bson:"_id, omitempty"`
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

func convertToContactResponse(result interface{}) (*AddContactResponse, error) {
	switch r := result.(type) {
	case []struct{ Key, Value string }:
		return convertFromStructSlice(r)
	case primitive.D:
		return convertFromPrimitiveD(r)
	default:
		return nil, fmt.Errorf("unsupported result type")
	}
}

func convertFromStructSlice(data []struct{ Key, Value string }) (*AddContactResponse, error) {
	response := &AddContactResponse{}
	for _, item := range data {
		switch item.Key {
		case "_id":
			response.ID = item.Value
		case "organization_id":
			response.OrganizationID = item.Value
		case "given_name":
			response.GivenName = item.Value
		case "last_name":
			response.LastName = item.Value
		case "email":
			response.Email = item.Value
		case "phone_number":
			response.PhoneNumber = item.Value
		}
	}
	return response, nil
}

func convertFromPrimitiveD(data primitive.D) (*AddContactResponse, error) {
	bytes, err := bson.Marshal(data)
	if err != nil {
		return nil, err
	}

	response := &AddContactResponse{}
	err = bson.Unmarshal(bytes, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (Mutation) AddContact(p graphql.ResolveParams, rbac rbac.RBAC, args AddContact) (*AddContactResponse, error) {
	c := Contact{}
	insertResult, err := db.Save(p.Context, &c, &args)
	if err != nil {
		return nil, err
	}
	filter := bson.M{"_id": insertResult.InsertedID}
	result, err := db.Find(p.Context, &c, filter)
	if err != nil {
		return nil, err
	}

	// Cast the result to *Contact
	response, err := convertToContactResponse(result)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return response, nil
}

func (Mutation) DeleteContact(p graphql.ResolveParams, rbac rbac.RBAC, filter ContactID) (bool, error) {
	c := Contact{}
	err := db.Delete(p.Context, &c, bson.M{"_id": filter.ID})
	return false, err
}

func (Mutation) UpdateContact(p graphql.ResolveParams, rbac rbac.RBAC, args EditContact) (Contact, error) {
	c := Contact{
		GivenName: args.GivenName,
		LastName: args.LastName,
		Email: args.Email,
		PhoneNumber: args.PhoneNumber,
		OrganizationID: args.OrganizationID,
	}
	err := db.Update(p.Context, &c, map[string]interface{}{
		"_id": args.ID,
	}, args)
	return c, err
}

package contacts

import (
	"errors"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/segmentio/ksuid"
	"neodeliver.com/engine/db"
	ggraphql "neodeliver.com/engine/graphql"
	"neodeliver.com/engine/rbac"
	utils "neodeliver.com/utils"
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
	Lang               *string   `bson:"lang" json:"lang"`
}

func (c ContactData) Validate() error {
	match := utils.ValidateEmail(c.Email)
	if !match {
		return errors.New("Email address is not valid")
	}
	match = utils.ValidatePhone(c.PhoneNumber)
	if !match {
		return errors.New("Phone number is not valid")
	}
	match = utils.ValidateLanguageCode(c.Lang)
	if !match {
		return errors.New("Language is not valid")
	}
	for _, token := range c.NotificationTokens {
		if !utils.ValidateNotificationToken(&token) {
			return errors.New("Notification tokens include invalid token")
		}
	}

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

type ContactEdit struct {
	ID   string
	Data ContactData	`json:"data" bson:"data"`
}

type TagAssign struct {
	ContactID	string
	TagID		string
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

	numberOfSameEmail, _ := db.Count(p.Context, &c, map[string]string{"organization_id": c.OrganizationID, "email": *args.Email})
	if numberOfSameEmail >= 1 {
		return c, errors.New("The email is already registered within your organization")
	}

	numberOfSameID, _ := db.Count(p.Context, &c, map[string]string{"organization_id": c.OrganizationID, "external_id": *args.ExternalID})
	if numberOfSameID >= 1 {
		return c, errors.New("The ID is duplicated within your organization")
	}

	_, err := db.Save(p.Context, &c)
	return c, err
}

func (Mutation) UpdateContact(p graphql.ResolveParams, rbac rbac.RBAC, args ContactEdit) (Contact, error) {
	if err := args.Data.Validate(); err != nil {
		return Contact{}, err
	}

	// only update the fields that were passed in params
	data := ggraphql.ArgToBson(p.Args["data"], args.Data)
	if len(data) == 0 {
		return Contact{}, errors.New("no data to update")
	}

	c := Contact{}

	if args.Data.Email != nil {
		numberOfSameEmail, _ := db.Count(p.Context, &c, map[string]string{"organization_id": rbac.OrganizationID, "email": *args.Data.Email})
		if numberOfSameEmail >= 1 {
			return c, errors.New("The email is duplicated within your organization")
		}
	}

	if args.Data.ExternalID != nil {
		numberOfSameID, _ := db.Count(p.Context, &c, map[string]string{"organization_id": rbac.OrganizationID, "external_id": *args.Data.ExternalID})
		if numberOfSameID >= 1 {
			return c, errors.New("The ID is duplicated within your organization")
		}
	}

	// Save the updated contact to the database
	err := db.Update(p.Context, &c, map[string]string{
		"_id": args.ID,
	}, data)

	return c, err
}

func (Mutation) DeleteContact(p graphql.ResolveParams, rbac rbac.RBAC, filter ContactID) (bool, error) {
	c := Contact{}
	err := db.Delete(p.Context, &c, map[string]string{"_id": filter.ID})
	return true, err
}

type ContactTag struct {
	ID			string	`bson:"_id"`
	ContactID	string	`bson:"contact_id" json:"contact_id"`
	TagID		string	`bson:"tag_id" json:"tag_id"`
}

func (Mutation) AssignTag(p graphql.ResolveParams, rbac rbac.RBAC, args TagAssign) (ContactTag, error) {
	r := ContactTag{
		ID:			"ctc_tag_" + ksuid.New().String(),
		ContactID:	args.ContactID,
		TagID:		args.TagID,
	}
	_, err := db.Save(p.Context, &r)
	return r, err
}

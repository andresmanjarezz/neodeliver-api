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
	"go.mongodb.org/mongo-driver/bson"
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
	Email              *string  `bson:"email, omitempty" json:"email, omitempty"`
	NotificationTokens []string `bson:"notification_tokens" json:"notification_tokens"`
	PhoneNumber        *string  `bson:"phone_number" json:"phone_number"`
	Lang               *string   `bson:"lang" json:"lang"`
}

func (c ContactData) Validate() error {
	if c.Email != nil {
		if !utils.ValidateEmail(c.Email) {
			return errors.New(utils.MessageEmailInvalid)
		}
	}
	if c.PhoneNumber != nil {
		if !utils.ValidatePhone(c.PhoneNumber) {
			return errors.New(utils.MessagePhoneNumberInvalid)
		}
	}
	if c.Lang != nil {
		if !utils.ValidateLanguageCode(c.Lang) {
			return errors.New(utils.MessageLangCodeInvalid)
		}
	}
	if c.NotificationTokens != nil {
		for _, token := range c.NotificationTokens {
			if !utils.ValidateNotificationToken(&token) {
				return errors.New(utils.MessageNotificationTokenInvalid)
			}
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
	Tags	   	   []string	 `bson:"tags"`
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
		Status:         utils.ContactStatusActive,
		SubscribedAt:   time.Now(),
		ContactData:    args,
		Tags:			make([]string, 0),
	}

	if err := c.Validate(); err != nil {
		return c, err
	}

	filter := bson.M{
		"organization_id": c.OrganizationID,
		"$or": []bson.M{
			{"email": *args.Email},
			{"external_id": *args.ExternalID},
		},
	}

	numberOfDuplicates, err := db.Count(p.Context, &c, filter)
	if err != nil {
		return c, errors.New(utils.MessageDefaultError)
	}
	if numberOfDuplicates >= 1 {
		return c, errors.New(utils.MessageDuplicationError)
	}

	_, err = db.Save(p.Context, &c)
	if err != nil {
		return c, errors.New(utils.MessageDefaultError)
	}

	return c, nil
}

func (Mutation) UpdateContact(p graphql.ResolveParams, rbac rbac.RBAC, args ContactEdit) (Contact, error) {
	args.Data = utils.FilterNilFields(args.Data).(ContactData)
	if err := args.Data.Validate(); err != nil {
		return Contact{}, err
	}

	// only update the fields that were passed in params
	data := ggraphql.ArgToBson(p.Args["data"], args.Data)
	if len(data) == 0 {
		return Contact{}, errors.New(utils.MessageNoUpdateError)
	}

	c := Contact{}
	filter := bson.M{}
	if args.Data.Email != nil && args.Data.ExternalID != nil {
		filter = bson.M{
			"$or": []bson.M{
				{"email": *args.Data.Email},
				{"external_id": *args.Data.ExternalID},
			},
		}
	} else if args.Data.Email == nil && args.Data.ExternalID != nil {
		filter = bson.M{"external_id": *args.Data.ExternalID}
	} else if args.Data.ExternalID == nil && args.Data.Email != nil {
		filter = bson.M{"email": *args.Data.Email}
	}
	duplicateFilter := bson.M{
		"$and": []bson.M{
			{"organization_id": rbac.OrganizationID},
			{
				"_id": bson.M{
					"$not": bson.M{
						"$eq": args.ID,
					},
				},
			},
			filter,
		},
	}
	numberOfDuplicates, err := db.Count(p.Context, &c, duplicateFilter)
	if err != nil {
		return c, errors.New(utils.MessageDefaultError)
	}
	if numberOfDuplicates >= 1 {
		return c, errors.New(utils.MessageDuplicationError)
	}
	// Save the updated contact to the database
	err = db.Update(p.Context, &c, map[string]string{
		"_id": args.ID,
	}, data)

	return c, nil
}

func (Mutation) DeleteContact(p graphql.ResolveParams, rbac rbac.RBAC, filter ContactID) (bool, error) {
	c := Contact{}
	err := db.Delete(p.Context, &c, map[string]string{"_id": filter.ID})
	return true, err
}

func (Mutation) AssignTag(p graphql.ResolveParams, rbac rbac.RBAC, args TagAssign) (Contact, error) {
	c := Contact{}
	err := db.Find(p.Context, &c, map[string]string{
		"_id": args.ContactID,
	})
	if err != nil {
		return c, errors.New(utils.MessageContactCannotFindError)
	}

	t := Tag{}
	err = db.Find(p.Context, &t, map[string]string{
		"_id": args.TagID,
	})
	if err != nil {
		return c, errors.New(utils.MessageTagCannotFindError)
	}

	count := 0
	for _, tagID := range c.Tags {
		if tagID == args.TagID {
			count ++
		}
	}
	if count != 0 {
		return c, errors.New(utils.MessageTagAssignDuplicationError)
	}

	c.Tags = append(c.Tags, args.TagID)
	err = db.Update(p.Context, &c, map[string]string{
		"_id": args.ContactID,
	}, c)
	if err != nil {
		return c, errors.New(utils.MessageTagCannotAssignError)
	}

	return c, err
}

func (Mutation) UnassignTag(p graphql.ResolveParams, rbac rbac.RBAC, args TagAssign) (Contact, error) {
	c := Contact{}
	err := db.Find(p.Context, &c, map[string]string{
		"_id": args.ContactID,
	})
	if err != nil {
		return c, errors.New(utils.MessageContactCannotFindError)
	}

	t := Tag{}
	err = db.Find(p.Context, &t, map[string]string{
		"_id": args.TagID,
	})
	if err != nil {
		return c, errors.New(utils.MessageTagCannotFindError)
	}

	updateTags := make([]string, 0)
	for _, tagID := range c.Tags {
		if tagID != args.TagID {
			updateTags = append(updateTags, tagID)
		}
	}
	if len(updateTags) == len(c.Tags) {
		return c, errors.New(utils.MessageTagNotAssignedError)
	}

	c.Tags = updateTags
	err = db.Update(p.Context, &c, map[string]string{
		"_id": args.ContactID,
	}, c)
	if err != nil {
		return c, errors.New(utils.MessageTagCannotAssignError)
	}

	return c, err
}

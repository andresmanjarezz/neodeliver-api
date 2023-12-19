package contacts

import (
	"errors"
	"time"
	"encoding/base64"
	"encoding/csv"
	"strings"

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

	return nil
}

type Contact struct {
	ID             string    `json:"id" bson:"_id,omitempty"`
	OrganizationID string    `bson:"organization_id"`
	Status         string    `bson:"status" json:"status"`
	SubscribedAt   time.Time `bson:"subscribed_at" json:"subscribed_at"`
	ContactData    `bson:",inline" json:",inline"`
	Tags	   	   []string	 `bson:"tags"`
	ContactStats   ContactStats	`bson:"contact_stats" json:"contact_stats"`
}

type ContactID struct {
	ID string `bson:"_id, omitempty"`
}

type ContactEdit struct {
	ID   string
	Data ContactData	`json:"data"`
}

type TagAssign struct {
	ContactID	string
	TagID		string
}

func (Mutation) CreateContact(p graphql.ResolveParams, rbac rbac.RBAC, args ContactData) (Contact, error) {
	contactStatsItem := ContactStatsItem{
		CampaignsSent:		0,
		LastCampaignSent:	time.Now(),
		MessagesOpened:		0,
		LastMessageOpened:	time.Now(),
		MessagesClicked:	0,
		LastMessageClicked:	time.Now(),
	}
	c := Contact{
		ID:             "ctc_" + ksuid.New().String(),
		OrganizationID: rbac.OrganizationID,
		Status:         utils.ContactStatusActive,
		SubscribedAt:   time.Now(),
		ContactData:    args,
		Tags:			make([]string, 0),
		ContactStats:	ContactStats{
			Email:			contactStatsItem,
			SMS:			contactStatsItem,
			Notifications:	contactStatsItem,
		},
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
		utils.LogErrorToSentry(err)
		return c, errors.New(utils.MessageDefaultError)
	}
	if numberOfDuplicates >= 1 {
		return c, errors.New(utils.MessageDuplicationError)
	}

	_, err = db.Save(p.Context, &c)
	if err != nil {
		utils.LogErrorToSentry(err)
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
			"organization_id": rbac.OrganizationID,
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
		utils.LogErrorToSentry(err)
		return c, errors.New(utils.MessageDefaultError)
	}
	if numberOfDuplicates >= 1 {
		return c, errors.New(utils.MessageDuplicationError)
	}
	// Save the updated contact to the database
	err = db.Update(p.Context, &c, map[string]string{
		"_id": args.ID,
	}, data)
	if err != nil {
		utils.LogErrorToSentry(err)
		return c, errors.New(utils.MessageDefaultError)
	}

	return c, nil
}

func (Mutation) DeleteContact(p graphql.ResolveParams, rbac rbac.RBAC, filter ContactID) (bool, error) {
	c := Contact{}
	err := db.Delete(p.Context, &c, map[string]string{"_id": filter.ID})
	if err != nil {
		utils.LogErrorToSentry(err)
		return false, errors.New(utils.MessageContactCannotDeleteError)
	}
	return true, nil
}

func (Mutation) AssignTag(p graphql.ResolveParams, rbac rbac.RBAC, args TagAssign) (Contact, error) {
	c := Contact{}
	err := db.Find(p.Context, &c, map[string]string{
		"_id": args.ContactID,
	})
	if err != nil {
		utils.LogErrorToSentry(err)
		return c, errors.New(utils.MessageContactCannotFindError)
	}

	t := Tag{}
	err = db.Find(p.Context, &t, map[string]string{
		"_id": args.TagID,
	})
	if err != nil {
		utils.LogErrorToSentry(err)
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
		utils.LogErrorToSentry(err)
		return c, errors.New(utils.MessageTagCannotAssignError)
	}

	return c, err
}

type ContactCSVFile struct {
	Base64Content		string	`json:"base64_content"`
}

func (Mutation) CreateContactsFromCSV(p graphql.ResolveParams, rbac rbac.RBAC, args ContactCSVFile) ([]Contact, error) {
	decodedData, err := base64.StdEncoding.DecodeString(args.Base64Content)
	if err != nil {
		return make([]Contact, 0), errors.New(utils.MessageDefaultError)
	}
	
	reader := csv.NewReader(strings.NewReader(string(decodedData)))

	records, err := reader.ReadAll()
	if err != nil {
		return make([]Contact, 0), errors.New(utils.MessageDefaultError)
	}

	contactStatsItem := ContactStatsItem{
		CampaignsSent:		0,
		LastCampaignSent:	time.Now(),
		MessagesOpened:		0,
		LastMessageOpened:	time.Now(),
		MessagesClicked:	0,
		LastMessageClicked:	time.Now(),
	}
	contacts := make([]Contact, len(records) - 1)
	added_contacts := make([]Contact, 0)
	for i, record := range records[1:] {
		contacts[i] = Contact{
			ID:             "ctc_" + ksuid.New().String(),
			OrganizationID: rbac.OrganizationID,
			Status:         utils.ContactStatusActive,
			SubscribedAt:   time.Now(),
			ContactData:    ContactData{
				GivenName:		&record[0],
				LastName:		&record[1],
				Email:			&record[2],
				ExternalID:		&record[3],
				PhoneNumber:	&record[4],
				Lang:			&record[5],
			},
			Tags:		 	make([]string, 0),
			ContactStats:	ContactStats{
				Email:			contactStatsItem,
				SMS:			contactStatsItem,
				Notifications:	contactStatsItem,
			},
		}
		if err := contacts[i].ContactData.Validate(); err != nil {
			continue
		}

		filter := bson.M{
			"$or": []bson.M{
				{"email": contacts[i].ContactData.Email},
				{"external_id": contacts[i].ContactData.ExternalID},
			},
		}
	
		numberOfDuplicates, err := db.Count(p.Context, &contacts[i], filter)
		if err != nil {
			continue
		}
		if numberOfDuplicates >= 1 {
			continue
		}
	
		_, err = db.Save(p.Context, &contacts[i])
		if err != nil {
			continue
		}
		added_contacts = append(added_contacts, contacts[i])
	}
	return added_contacts, nil
}

func (Mutation) UnassignTag(p graphql.ResolveParams, rbac rbac.RBAC, args TagAssign) (Contact, error) {
	c := Contact{}
	err := db.Find(p.Context, &c, map[string]string{
		"_id": args.ContactID,
	})
	if err != nil {
		utils.LogErrorToSentry(err)
		return c, errors.New(utils.MessageContactCannotFindError)
	}

	t := Tag{}
	err = db.Find(p.Context, &t, map[string]string{
		"_id": args.TagID,
	})
	if err != nil {
		utils.LogErrorToSentry(err)
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
		utils.LogErrorToSentry(err)
		return c, errors.New(utils.MessageTagCannotAssignError)
	}

	return c, err
}

func (Mutation) GetContactsByTag(p graphql.ResolveParams, rbac rbac.RBAC, args TagID) ([]Contact, error) {
	c := Contact{}
	contacts := make([]Contact, 0)
	filter := bson.M{
        "tags": bson.M{
            "$in": bson.A{args.ID},
        },
		"organization_id": rbac.OrganizationID,
    }
	err := db.FindAll(p.Context, &c, &contacts, filter)
	if err != nil {
		return []Contact{}, errors.New(utils.MessageDefaultError)
	}
	return contacts, err
}

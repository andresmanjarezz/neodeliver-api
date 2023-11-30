package campaigns

import (
	"time"
	"errors"
	
	"github.com/graphql-go/graphql"
	"github.com/segmentio/ksuid"
	ggraphql "neodeliver.com/engine/graphql"
	"neodeliver.com/engine/db"
	"neodeliver.com/engine/rbac"
	"neodeliver.com/utils"
)

type EmailCampaign struct {
	ID            	string `json:"id" bson:"_id"`
	OrganizationID	string `json:"organization_id" bson:"organization_id"`
	Draft          	bool   `json:"draft" bson:"draft"`
	CreatedAt      	time.Time `json:"created_at"`
	EmailCampaignData	`json:",inline" bson:",inline"`
}

type EmailCampaignData struct {
	Sender			string `json:"sender" json:"sender"`
	Name			string `json:"name" bson:"name"`
	Scheduled		bool   `json:"scheduled" bson:"scheduled"`
	DeliveryTime	time.Time `json:"delivery_time" bson:"delivery_time"`
}

type EmailCampaignEdit struct {
	ID			string	`json:"id"`
	Data		EmailCampaignData	`json:"data"`
}

type EmailCampaignID struct {
	ID			string	`json:"id"`
}

type EmailCampaignSetting struct {
	TrackingOption		string	`json:"tracking_option" bson:"tracking_option"`
}

func (Mutation) CreateEmailCampaign(p graphql.ResolveParams, rbac rbac.RBAC, args EmailCampaignData) (EmailCampaign, error) {
	e := EmailCampaign{
		ID:				"ecmp_" + ksuid.New().String(),
		OrganizationID:	rbac.OrganizationID,
		CreatedAt:		time.Now(),
		EmailCampaignData:	args,
	}

	_, err := db.Save(p.Context, &e)
	return e, err
}

func (Mutation) UpdateEmailCampaign(p graphql.ResolveParams, rbac rbac.RBAC, args EmailCampaignEdit) (EmailCampaign, error) {
	data := ggraphql.ArgToBson(p.Args["data"], args.Data)
	if len(data) == 0 {
		return EmailCampaign{}, errors.New(utils.MessageDefaultError)
	}

	e := EmailCampaign{}
	err := db.Update(p.Context, &e, map[string]string{
		"_id": args.ID,
	}, data)
	if err != nil {
		return e, errors.New(utils.MessageDefaultError)
	}

	return e, nil
}

func (Mutation) DeleteEmailCampaign(p graphql.ResolveParams, rbac rbac.RBAC, filter EmailCampaignID) (bool, error) {
	e := EmailCampaign{}
	err := db.Delete(p.Context, &e, map[string]string{"_id": filter.ID})
	if err != nil {
		return false, errors.New(utils.MessageCampaignCannotDeleteError)
	}
	return true, nil
}

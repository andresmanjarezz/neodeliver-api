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

type NotificationCampaign struct {
	ID            	string `json:"id" bson:"_id"`
	OrganizationID	string `json:"organization_id" bson:"organization_id"`
	Draft          	bool   `json:"draft" bson:"draft"`
	CreatedAt      	time.Time `json:"created_at"`
	NotificationCampaignData	`json:",inline" bson:",inline"`
}

type NotificationCampaignData struct {
	Platform		string `json:"platform" bson:"platform"`
	Name			string `json:"name" bson:"name"`
	Scheduled		bool   `json:"scheduled" bson:"scheduled"`
	DeliveryTime	time.Time `json:"delivery_time" bson:"delivery_time"`
	Settings		NotificationCampaignSetting	`json:"settings" bson:"settings"`
}

type NotificationCampaignEdit struct {
	ID			string	`json:"id"`
	Data		NotificationCampaignData	`json:"data"`
}

type NotificationCampaignID struct {
	ID			string	`json:"id"`
}

type NotificationCampaignSetting struct {
	DeeplinkUrl			string	`json:"deeplink_url" bson:"deeplink_url"`
	TrackingOption		string	`json:"tracking_option" bson:"tracking_option"`
	ActionAfterClicked	string	`json:"action_after_clicked" bson:"action_after_clicked"`
}

func (Mutation) CreateNotificationCampaign(p graphql.ResolveParams, rbac rbac.RBAC, args NotificationCampaignData) (NotificationCampaign, error) {
	n := NotificationCampaign{
		ID:				"ncmp_" + ksuid.New().String(),
		OrganizationID:	rbac.OrganizationID,
		CreatedAt:		time.Now(),
		NotificationCampaignData: args,
	}

	_, err := db.Save(p.Context, &n)

	return n, err
}

func (Mutation) UpdateNotificationCampaign(p graphql.ResolveParams, rbac rbac.RBAC, args NotificationCampaignEdit) (NotificationCampaign, error) {
	data := ggraphql.ArgToBson(p.Args["data"], args.Data)
	if len(data) == 0 {
		return NotificationCampaign{}, errors.New(utils.MessageDefaultError)
	}

	e := NotificationCampaign{}
	err := db.Update(p.Context, &e, map[string]string{
		"_id": args.ID,
	}, data)
	if err != nil {
		return e, errors.New(utils.MessageDefaultError)
	}

	return e, nil
}

func (Mutation) DeleteNotificationCampaign(p graphql.ResolveParams, rbac rbac.RBAC, filter NotificationCampaignID) (bool, error) {
	e := NotificationCampaign{}
	err := db.Delete(p.Context, &e, map[string]string{"_id": filter.ID})
	if err != nil {
		return false, errors.New(utils.MessageCampaignCannotDeleteError)
	}
	return true, nil
}

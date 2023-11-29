package campaigns

import (
	"time"
	
	"github.com/graphql-go/graphql"
	"github.com/segmentio/ksuid"
	"neodeliver.com/engine/db"
	"neodeliver.com/engine/rbac"
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

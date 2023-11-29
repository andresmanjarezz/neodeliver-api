package campaigns

import (
	"time"
	
	"github.com/graphql-go/graphql"
	"github.com/segmentio/ksuid"
	"neodeliver.com/engine/db"
	"neodeliver.com/engine/rbac"
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

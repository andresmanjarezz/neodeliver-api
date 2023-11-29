package campaigns

import "time"

type Campaign struct {
	ID             string `bson:"_id" json:"id"`
	OrganizationID string `bson:"organization_id" json:"organization_id"`
	Draft          bool   `bson:"draft" json:"draft"`
	CreatedAt      time.Time `json:"created_at"`
	CampaignType   string  `bson:"campaign_type" json:"campaign_type"`
	CampaignData   `bson:",inline" json:",inline"`
}

type CampaignSetting struct {
	DeeplinkUrl		string	`bson:"deeplink_url" json:"deeplink_url"`
	TrackingOption	string	`bson:"tracking_option" json:"tracking_option"`
	ActionAfterClicked	string	`bson:"action_after_clicked" json:"action_after_clicked"`
}

type CampaignData struct {
	Name		   string `bson:"name" json:"name"`
	Recipients	   []string `bson:"recipients" json:"recipients"`
	Scheduled	   bool   `bson:"scheduled" json:"scheduled"`
	DeliveryTime   time.Time `bson:"delivery_time" json:"delivery_time"`
	Transactional  bool   `bson:"transactional" json:"transactional"`
	Settings	   CampaignSetting `bson:"settings" json:"settings"`
}

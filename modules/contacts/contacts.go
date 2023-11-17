package contacts

import "time"

type Contact struct {
	ID                 string `bson:"_id"`
	OrganizationID     string `bson:"organization_id"`
	GivenName          string
	LastName           string
	Email              string
	NotificationTokens []string
	PhoneNumber        string
	Status             string
	SubscribedAt       time.Time
	Lang               string
	// Meta               map[string]interface{}
	// Tags 						 []string
	// SubscribedChannels []string
	Stats ContactStats
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

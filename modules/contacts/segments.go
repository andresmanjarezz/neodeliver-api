package contacts

import (
	"time"

	"github.com/graphql-go/graphql"
	"neodeliver.com/engine/rbac"
	"neodeliver.com/engine/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Segment struct {
	ID             string `bson:"_id,omitempty" json:"id"`
	OrganizationID string `bson:"organization_id" json:"organization_id"`
	Name           string `bson:"name" json:"name"`
	Filters		   string `bson:"filters" json:"filters"`
	Subscription   int	  `bson:"subscription" json:"subscription"`
	OpensCount	   int    `bson:"opens_count" json:"opens_count"`
	ClickRate	   int	  `bson:"click_rate" json:"click_rate"`
	MailsSentCount int	  `bson:"mails_sent_count" json:"mail_sent_count"`
	CreatedAt      time.Time `bson:"created_at" json:"created_at"`
}

type CreateSegment struct {
	OrganizationID string `bson:"organization_id" json:"organization_id"`
	Name           string `bson:"_id,omitempty" json:"id"`
	Filters		   string `bson:"filters" json:"filters"`
}

type SegmentID struct {
	ID			   string `bson:"_id, omitempty"`
}

func (Mutation) CreateSegment(p graphql.ResolveParams, rbac rbac.RBAC, args CreateSegment) (*Segment, error) {
	s := Segment{
		Name:			args.Name,
		OrganizationID:	args.OrganizationID,
		Filters:		args.Filters,
		Subscription:	0,
		OpensCount:		0,
		ClickRate:		0,
		MailsSentCount:	0,
		CreatedAt:		time.Now(),
	}

	insertResult, err := db.Save(p.Context, &s)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"_id": insertResult.InsertedID}
	_, err = db.Find(p.Context, &s, filter)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (Mutation) UpdateSegment(p graphql.ResolveParams, rbac rbac.RBAC, args Segment) (*Segment, error) {
	s := Segment{
		Name:			args.Name,
		OrganizationID:	args.OrganizationID,
		Filters:		args.Filters,
		Subscription:	args.Subscription,
		OpensCount:		args.OpensCount,
		ClickRate:		args.ClickRate,
		MailsSentCount:	args.MailsSentCount,
	}

	objectID, _ := primitive.ObjectIDFromHex(args.ID)
	filter := bson.M{"_id": objectID}

	err := db.Update(p.Context, &s, filter, &s)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (Mutation) DeleteSegment(p graphql.ResolveParams, rbac rbac.RBAC, filter SegmentID) (bool, error) {
	s := Segment{}
	objectID, _ := primitive.ObjectIDFromHex(filter.ID)
	err := db.Delete(p.Context, &s, bson.M{"_id": objectID})
	if err != nil {
		return false, err
	}
	return true, err
}

// func (Mutation) SendNewMail(p graphql.ResolveParams, rbac rbac.RBAC, filter SegmentID) (*Segment, error) {
// 	s := Segment{}
// 	objectID, err := primitive.ObjectIDFromHex(filter.ID)
// 	t, _ := db.Find(p.Context, &s, bson.M{"_id": objectID})
// 	fmt.Println(t.(Segment))
// 	return &s, err
// }

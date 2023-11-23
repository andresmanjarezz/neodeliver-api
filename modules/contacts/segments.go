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

type Segment struct {
	ID             string `bson:"_id,omitempty" json:"id"`
	OrganizationID string    `bson:"organization_id"`
	OpensCount	   int    `bson:"opens_count" json:"opens_count"`
	ClickRate	   int	  `bson:"click_rate" json:"click_rate"`
	MailsSentCount int	  `bson:"mails_sent_count" json:"mail_sent_count"`
	CreatedAt      time.Time `bson:"created_at" json:"created_at"`
	SegmentData			`bson:",inline" json:",inline"`
}

type SegmentData struct {
	Name           *string `bson:"name" json:"name"`
	Filters		   *string `bson:"filters" json:"filters"`
	Subscription   *int	  `bson:"subscription" json:"subscription"`
}

type SegmentID struct {
	ID			   string `bson:"_id, omitempty"`
}

func (s SegmentData) Validate() error {
	match := utils.ValidateMongoDBQuery(s.Filters)
	if !match {
		return errors.New("Filter query is not valid")
	}
	return nil
}

func (Mutation) CreateSegment(p graphql.ResolveParams, rbac rbac.RBAC, args SegmentData) (Segment, error) {
	s := Segment{
		ID:				"sgt_" + ksuid.New().String(),
		OrganizationID:	rbac.OrganizationID,
		OpensCount:		0,
		ClickRate:		0,
		MailsSentCount:	0,
		CreatedAt:		time.Now(),
		SegmentData:	args,
	}

	if err := s.Validate(); err != nil {
		return s, err
	}

	_, err := db.Save(p.Context, &s)

	return s, err
}

type SegmentEdit struct {
	ID		string
	Data	SegmentData	`json:"data"`
}

func (Mutation) UpdateSegment(p graphql.ResolveParams, rbac rbac.RBAC, args SegmentEdit) (Segment, error) {
	if err := args.Data.Validate(); err != nil {
		return Segment{}, err
	}
	// only update the fields that were passed in params
	data := ggraphql.ArgToBson(p.Args["data"], args.Data)
	if len(data) == 0 {
		return Segment{}, errors.New("no data to update")
	}

	s := Segment{}

	err := db.Update(p.Context, &s, map[string]string{
		"_id": args.ID,
	}, data)

	return s, err
}

func (Mutation) DeleteSegment(p graphql.ResolveParams, rbac rbac.RBAC, filter SegmentID) (bool, error) {
	s := Segment{}
	err := db.Delete(p.Context, &s, map[string]string{"_id": filter.ID})
	return true, err
}

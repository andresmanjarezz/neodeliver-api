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
	CreatedAt      time.Time `bson:"created_at" json:"created_at"`
	SegmentData			`bson:",inline" json:",inline"`
	SegmentStats		`bson:",inline" json:",inline"`
}

type SegmentStats struct {
	OpensCount	   int    `bson:"opens_count" json:"opens_count"`
	ClickRate	   int	  `bson:"click_rate" json:"click_rate"`
	MailsSentCount int	  `bson:"mails_sent_count" json:"mail_sent_count"`
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
	_, err := utils.ConvertQueryToBSON(*s.Filters)
	if err != nil {
		return errors.New(utils.MessageSegmentQueryInvalid)
	}
	return nil
}

func (Mutation) CreateSegment(p graphql.ResolveParams, rbac rbac.RBAC, args SegmentData) (Segment, error) {
	s := Segment{
		ID:				"sgt_" + ksuid.New().String(),
		OrganizationID:	rbac.OrganizationID,
		SegmentStats:	SegmentStats{
			OpensCount:		0,
			ClickRate:		0,
			MailsSentCount:	0,
		},
		CreatedAt:		time.Now(),
		SegmentData:	args,
	}
	
	utils.RemoveSpaces(s.Filters)
	if err := s.Validate(); err != nil {
		return s, err
	}

	numberOfDuplicates, err := db.Count(p.Context, &s, map[string]string{
		"name": *args.Name,
	})
	if err != nil {
		utils.LogErrorToSentry(err)
		return s, errors.New(utils.MessageDefaultError)
	}
	if numberOfDuplicates >= 1 {
		return s, errors.New(utils.MessageSegmentNameDuplicationError)
	}

	bsonObj, err := utils.ConvertQueryToBSON(*args.Filters)
	if err != nil {
		return s, err
	}

	if utils.GetQueryBSONDepth(bsonObj) > utils.SegmentMaximumQueryBSONDepthNumber {
		return s, errors.New(utils.MessageSegmentQueryDepthExceedError)
	}

	_, err = db.Save(p.Context, &s)
	if err != nil {
		utils.LogErrorToSentry(err)
		return s, errors.New(utils.MessageDefaultError)
	}

	return s, nil
}

type SegmentEdit struct {
	ID		string
	Data	SegmentData	`json:"data"`
}

func (Mutation) UpdateSegment(p graphql.ResolveParams, rbac rbac.RBAC, args SegmentEdit) (Segment, error) {
	// Validate the data before updating the segment
	if err := args.Data.Validate(); err != nil {
		return Segment{}, err
	}
	
	// Convert the data to BSON for updating only the specified fields
	data := ggraphql.ArgToBson(p.Args["data"], args.Data)
	if len(data) == 0 {
		return Segment{}, errors.New(utils.MessageNoUpdateError)
	}
	
	s := Segment{}
	
	// Update the segment in the database based on the provided ID and data
	err := db.Update(p.Context, &s, map[string]string{
		"_id": args.ID,
	}, data)
	if err != nil {
		utils.LogErrorToSentry(err)
		return s, errors.New(utils.MessageDefaultError)
	}
	
	return s, nil
}

func (Mutation) DeleteSegment(p graphql.ResolveParams, rbac rbac.RBAC, filter SegmentID) (bool, error) {
	s := Segment{}
	// Delete the segment from the database based on the provided filter ID
	err := db.Delete(p.Context, &s, map[string]string{"_id": filter.ID})
	if err != nil {
		utils.LogErrorToSentry(err)
		return false, errors.New(utils.MessageSegmentCannotDeleteError)
	}
	return true, nil
}

// GetContactsBySegmentQuery retrieves contacts based on a segment query
func (Mutation) GetContactsBySegmentQuery(p graphql.ResolveParams, rbac rbac.RBAC, args SegmentID) ([]Contact, error) {
	s := Segment{}
	// Find the segment by ID
	err := db.Find(p.Context, &s, map[string]string{
		"_id": args.ID,
	})
	if err != nil {
		return []Contact{}, errors.New(utils.MessageSegmentCannotFindError)
	}

	// Convert the segment filters to BSON
	bsonObj, err := utils.ConvertQueryToBSON(*s.Filters)
	if err != nil {
		return []Contact{}, err
	}

	c := Contact{}
	contacts := []Contact{}
	// Find all contacts matching the segment filters
	err = db.FindAll(p.Context, &c, &contacts, bsonObj)
	if err != nil {
		return []Contact{}, errors.New(utils.MessageDefaultError)
	}

	return contacts, err
}

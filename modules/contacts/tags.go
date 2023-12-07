package contacts

import (
	"errors"
	"time"

	"github.com/graphql-go/graphql"
	"go.mongodb.org/mongo-driver/bson"
	"neodeliver.com/engine/rbac"
	"neodeliver.com/engine/db"
	"github.com/segmentio/ksuid"
	ggraphql "neodeliver.com/engine/graphql"
	utils "neodeliver.com/utils"
)

type Tag struct {
	ID             string `bson:"_id,omitempty" json:"id"`
	OrganizationID string `bson:"organization_id"`
	ContactsCount  int       `bson:"contacts_count"`
	CreatedAt      time.Time `bson:"created_at" json:"created_at"`
	TagData		   `bson:",inline" json:",inline"`
}

type TagData struct {
	Name			*string	`bson:"name" json:"name"`
	Description		*string	`bson:"description" json:"description"`
}

type TagEdit struct {
	ID		string
	Data	TagData	`json:"data"`
}

type TagID struct {
	ID             string `bson:"_id, omitempty"`
}

func (Mutation) CreateTag(p graphql.ResolveParams, rbac rbac.RBAC, args TagData) (Tag, error) {
	t := Tag{
		ID:				"tag_" + ksuid.New().String(),
		OrganizationID:	rbac.OrganizationID,
		ContactsCount:	0,
		CreatedAt:      time.Now(),
		TagData:		args,
	}

	numberOfDuplicates, err := db.Count(p.Context, &t, map[string]string{"organization_id": t.OrganizationID, "name": *args.Name})
	if err != nil {
		utils.LogErrorToSentry(err)
		return t, errors.New(utils.MessageDefaultError)
	}
	if numberOfDuplicates >= 1 {
		return t, errors.New(utils.MessageTagNameDuplicationError)
	}

	_, err = db.Save(p.Context, &t)
	if err != nil {
		utils.LogErrorToSentry(err)
		return t, errors.New(utils.MessageDefaultError)

	}
	return t, nil
}

func (Mutation) UpdateTag(p graphql.ResolveParams, rbac rbac.RBAC, args TagEdit) (Tag, error) {
	// only update the fields that were passed in params
	data := ggraphql.ArgToBson(p.Args["data"], args.Data)
	if len(data) == 0 {
		return Tag{}, errors.New(utils.MessageNoUpdateError)
	}
	
	t := Tag{}

	filter := bson.M{
		"$and": []bson.M{
			{"organization_id": rbac.OrganizationID},
			{
				"_id": bson.M{
					"$not": bson.M{
						"$eq": args.ID,
					},
				},
			},
			{"name": *args.Data.Name},
		},
	}
	if args.Data.Name != nil {
		sameNameCount, err := db.Count(p.Context, &t, filter)
		if err != nil {
			utils.LogErrorToSentry(err)
			return t, errors.New(utils.MessageDefaultError)
		}
		if sameNameCount >= 1 {
			return t, errors.New(utils.MessageTagNameDuplicationError)
		}
	}

	err := db.Update(p.Context, &t, map[string]string{
		"_id": args.ID,
	}, data)
	if err != nil {
		utils.LogErrorToSentry(err)
		return t, errors.New(utils.MessageDefaultError)
	}

	return t, nil
}

func (Mutation) DeleteTag(p graphql.ResolveParams, rbac rbac.RBAC, filter TagID) (bool, error) {
	t := Tag{}
	err := db.Delete(p.Context, &t, map[string]string{"_id": filter.ID})
	if err != nil {
		utils.LogErrorToSentry(err)
		return false, errors.New(utils.MessageTagCannotDeleteError)
	}
	return true, nil
}

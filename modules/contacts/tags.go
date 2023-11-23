package contacts

import (
	"errors"
	"time"

	"github.com/graphql-go/graphql"
	"neodeliver.com/engine/rbac"
	"neodeliver.com/engine/db"
	"github.com/segmentio/ksuid"
	ggraphql "neodeliver.com/engine/graphql"
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

func (Mutation) AddTag(p graphql.ResolveParams, rbac rbac.RBAC, args TagData) (Tag, error) {
	t := Tag{
		ID:				"tag_" + ksuid.New().String(),
		OrganizationID:	rbac.OrganizationID,
		ContactsCount:	0,
		CreatedAt:      time.Now(),
		TagData:		args,
	}

	sameNameCount, _ := db.Count(p.Context, &t, map[string]string{"organization_id": t.OrganizationID, "name": *args.Name})
	if sameNameCount >= 1 {
		return t, errors.New("The name is already registered within your organization")
	}

	_, err := db.Save(p.Context, &t)
	return t, err
}

func (Mutation) UpdateTag(p graphql.ResolveParams, rbac rbac.RBAC, args TagEdit) (Tag, error) {
	// only update the fields that were passed in params
	data := ggraphql.ArgToBson(p.Args["data"], args.Data)
	if len(data) == 0 {
		return Tag{}, errors.New("no data to update")
	}
	
	t := Tag{}

	if args.Data.Name != nil {
		sameNameCount, _ := db.Count(p.Context, &t, map[string]string{"organization_id": t.OrganizationID, "name": *args.Data.Name})
		if sameNameCount >= 1 {
			return t, errors.New("The name is duplicated within your organization")
		}
	}

	err := db.Update(p.Context, &t, map[string]string{
		"_id": args.ID,
	}, data)

	return t, err
}

func (Mutation) DeleteTag(p graphql.ResolveParams, rbac rbac.RBAC, filter TagID) (bool, error) {
	t := Tag{}
	err := db.Delete(p.Context, &t, map[string]string{"_id": filter.ID})
	return true, err
}

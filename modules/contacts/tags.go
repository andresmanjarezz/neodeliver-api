package contacts

import (
	"time"

	"github.com/graphql-go/graphql"
	"neodeliver.com/engine/rbac"
	"neodeliver.com/engine/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Tag struct {
	ID             string `bson:"_id,omitempty" json:"id"`
	OrganizationID string `bson:"organization_id" json:"organization_id"`
	Name           string `bson:"name" json:"name"`
	ContactsCount  int       `bson:"contacts_count" json:"contacts_count"`
	CreatedAt      time.Time `bson:"created_at" json:"created_at"`
}

type AddTag struct {
	OrganizationID string `bson:"organization_id" json:"organization_id"`
	Name           string `bson:"name" json:"name"`
	ContactsCount  int    `bson:"contacts_count" json:"contacts_count"`
}

type TagID struct {
	ID             string `bson:"_id, omitempty"`
}

func (Mutation) AddTag(p graphql.ResolveParams, rbac rbac.RBAC, args AddTag) (*Tag, error) {
	t := Tag{
		Name:			args.Name,
		OrganizationID:	args.OrganizationID,
		ContactsCount:	args.ContactsCount,
		CreatedAt:      time.Now(),
	}

	insertResult, err := db.Save(p.Context, &t)
	if err != nil {
		return nil, err
	}
	
	filter := bson.M{"_id": insertResult.InsertedID}
	_, err = db.Find(p.Context, &t, filter)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (Mutation) UpdateTag(p graphql.ResolveParams, rbac rbac.RBAC, args Tag) (*Tag, error) {
	t := Tag{
		Name:			args.Name,
		OrganizationID:	args.OrganizationID,
		ContactsCount:	args.ContactsCount,
		CreatedAt:		args.CreatedAt,
	}
	
	objectID, _ := primitive.ObjectIDFromHex(args.ID)
	filter := bson.M{"_id": objectID}

	err := db.Update(p.Context, &t, filter, &t)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (Mutation) DeleteTag(p graphql.ResolveParams, rbac rbac.RBAC, filter TagID) (bool, error) {
	t := Tag{}
	objectID, _ := primitive.ObjectIDFromHex(filter.ID)
	err := db.Delete(p.Context, &t, bson.M{"_id": objectID})
	if err != nil {
		return false, err
	}
	return true, err
}

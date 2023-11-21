package contacts

import (
	"neodeliver.com/engine/graphql"
	"neodeliver.com/engine/rbac"
)

func Init(s *graphql.Builder) {
	// query single contact
	s.MongoQuery(Contact{}).Where(func(r rbac.RBAC, args graphql.ByID) map[string]interface{} {
		return map[string]interface{}{
			"_id":             args.ID,
			"organization_id": r.OrganizationID,
		}
	})

	// query contacts list
	s.MongoQuery([]Contact{}).Where(func(r rbac.RBAC) map[string]interface{} {
		return map[string]interface{}{
			"organization_id": r.OrganizationID,
		}
	})

	// query tags list
	s.MongoQuery([]Tag{}).Where(func(r rbac.RBAC) map[string]interface{} {
		return map[string]interface{}{
			"organization_id": r.OrganizationID,
		}
	})

	// query segments list
	s.MongoQuery([]Segment{}).Where(func(r rbac.RBAC) map[string]interface{} {
		return map[string]interface{}{
			"organization_id": r.OrganizationID,
		}
	})

	s.AddMutationMethods(Mutation{})
}

type Mutation struct{}

package campaigns

import (
	"neodeliver.com/engine/graphql"
	"neodeliver.com/engine/rbac"
)

func Init(s *graphql.Builder) {
	// query single campaign
	s.MongoQuery(Campaign{}).Where(func(r rbac.RBAC, args graphql.ByID) map[string]interface{} {
		return map[string]interface{}{
			"_id":             args.ID,
			"organization_id": r.OrganizationID,
		}
	})

	// query multiple campaign
	s.MongoQuery([]Campaign{}).Where(func(r rbac.RBAC) map[string]interface{} {
		return map[string]interface{}{
			"organization_id": r.OrganizationID,
		}
	})

	s.AddMutationMethods(Mutation{})
}

type Mutation struct{}

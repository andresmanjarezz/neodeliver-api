package campaigns

import (
	"time"
	"errors"
	
	"github.com/graphql-go/graphql"
	"github.com/segmentio/ksuid"
	ggraphql "neodeliver.com/engine/graphql"
	"neodeliver.com/engine/db"
	"neodeliver.com/engine/rbac"
	"neodeliver.com/utils"
)

type TransactionalMessage struct {
	ID            	string `json:"id" bson:"_id"`
	Draft          	bool   `json:"draft" bson:"draft"`
	CreatedAt      	time.Time `json:"created_at"`
	TransactionalMessageData	`json:",inline" bson:",inline"`
}

type TransactionalMessageData struct {
	Name			string `json:"name" bson:"name"`
	Status			string `json:"status" bson:"status"`
	Recipients		int	`json:"recipients" bson:"recipients"`
	Opens			int	`json:"opens" bson:"opens"`
	Clicks			int	`json:"clicks" bson:"clicks"`
	Unsubs			int	`json:"unsubs" bson:"unsubs"`
}

type TransactionalMessageFolder struct {
	ID			string	`json:"id" bson:"_id"`
	OrganizationID	string `json:"organization_id" bson:"organization_id"`
	Name		string	`json:"name" bson:"any"`
	CreatedAt	time.Time `json:"created_at"`
	TransactionalMessages	[]string	`json:"transactional_messages" bson:"transactional_messages"`
}

type TransactionalMessageInput	struct {
	Name		string	`json:"name"`
	FolderID	string	`json:"folder_id"`
}

type TransactionalObjectID struct {
	ID			string `json:"id"`
}

type TransactionalMessageFolderData struct {
	Name		string	`json:"name"`
}

type TransactionalMessageFolderEdit struct {
	ID			string	`json:"id"`
	Data		TransactionalMessageFolderData	`json:"data"`
}

type TransactionalMessageEdit struct {
	ID			string	`json:"id"`
	Data		TransactionalMessageData	`json:"data"`
}

func (Mutation) CreateTransactionalMessageFolder(p graphql.ResolveParams, rbac rbac.RBAC, args TransactionalMessageFolderData) (TransactionalMessageFolder, error) {
	f := TransactionalMessageFolder{
		ID:				"tfld_" + ksuid.New().String(),
		OrganizationID:	rbac.OrganizationID,
		Name:			args.Name,
		CreatedAt:		time.Now(),
		TransactionalMessages:	make([]string, 0),
	}

	_, err := db.Save(p.Context, &f)
	
	return f, err
}

func (Mutation) CreateTransactionalMessage(p graphql.ResolveParams, rbac rbac.RBAC, args TransactionalMessageInput) (TransactionalMessage, error) {
	m := TransactionalMessage{
		ID:				"tmsg_" + ksuid.New().String(),
		TransactionalMessageData:	TransactionalMessageData{
			Name:			args.Name,
			Status:			"",
			Recipients:		0,
			Opens:			0,
			Clicks:			0,
			Unsubs:			0,
		},
	}
	f := TransactionalMessageFolder{}
	err := db.Find(p.Context, &f, map[string]string{
		"_id": args.FolderID,
	})

	f.TransactionalMessages = append(f.TransactionalMessages, m.ID)
	err = db.Update(p.Context, &f, map[string]string{
		"_id": args.FolderID,
	}, f)

	if err != nil {
		return m, errors.New(utils.MessageTagCannotAssignError)
	}
	return m, nil
}

func (Mutation) DeleteTransactionalMessageFolder(p graphql.ResolveParams, rbac rbac.RBAC, filter TransactionalObjectID) (bool, error) {
	f := TransactionalMessageFolder{}
	err := db.Find(p.Context, &f, map[string]string{
		"_id": filter.ID,
	})
	if err != nil {
		return false, errors.New(utils.MessageTransactionalMessageFolderCannotDeleteError)
	}
	for _, messageID := range f.TransactionalMessages {
		m := TransactionalMessage{}
		err = db.Delete(p.Context, &m, map[string]string{"_id": messageID})
		if err != nil {
			return false, errors.New(utils.MessageTransactionalMessageFolderCannotDeleteError)
		}
	}
	err = db.Delete(p.Context, &f, map[string]string{"_id": filter.ID})
	if err != nil {
		return false, errors.New(utils.MessageTransactionalMessageFolderCannotDeleteError)
	}
	return true, nil
}

func (Mutation) DeleteTransactionalMessage(p graphql.ResolveParams, rbac rbac.RBAC, filter TransactionalObjectID) (bool, error) {
	m := TransactionalMessage{}
	err := db.Delete(p.Context, &m, map[string]string{"_id": filter.ID})
	if err != nil {
		return false, errors.New(utils.MessageTransactionalMessageCannotDeleteError)
	}
	return true, nil
}

func (Mutation) UpdateTransactionMessageFolder(p graphql.ResolveParams, rbac rbac.RBAC, args TransactionalMessageFolderEdit) (TransactionalMessageFolder, error) {
	f := TransactionalMessageFolder{}

	data := ggraphql.ArgToBson(p.Args["data"], args.Data)
	if len(data) == 0 {
		return f, errors.New(utils.MessageNoUpdateError)
	}

	err := db.Update(p.Context, &f, map[string]string{
		"_id": args.ID,
	}, data)
	if err != nil {
		return f, errors.New(utils.MessageDefaultError)
	}
	return f, nil
}

func (Mutation) UpdateTransactionalMessage(p graphql.ResolveParams, rbac rbac.RBAC, args TransactionalMessageEdit) (TransactionalMessage, error) {
	m := TransactionalMessage{}

	data := ggraphql.ArgToBson(p.Args["data"], args.Data)
	if len(data) == 0 {
		return m, errors.New(utils.MessageNoUpdateError)
	}

	err := db.Update(p.Context, &m, map[string]string{
		"_id": args.ID,
	}, data)
	if err != nil {
		return m, errors.New(utils.MessageDefaultError)
	}
	return m, nil
}

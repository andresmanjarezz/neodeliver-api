package db

import (
	"context"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/gertd/go-pluralize"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

func Client() *mongo.Database {
	if client == nil {
		uri := os.Getenv("mongodb")
		// fmt.Println(string(uri))

		serverAPI := options.ServerAPI(options.ServerAPIVersion1)
		opts := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI)
		// Create a new client and connect to the server
		c, err := mongo.Connect(context.TODO(), opts)
		if err != nil {
			panic(err)
		}

		client = c
	}

	return client.Database("neodeliver")
}

func Close() {
	if client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := client.Disconnect(ctx); err != nil {
			panic(err)
		}

		client = nil
	}
}

// ---

func Find(ctx context.Context, o interface{}, filter interface{}, opts ...*options.FindOneOptions) error {
	c := Client()
	return c.Collection(CollectionName(o)).FindOne(ctx, filter, opts...).Decode(o)
}

// func Save(ctx context.Context, o interface{}) error {
// 	c := Client().Collection(CollectionName(o))
// 	_, err := c.InsertOne(ctx, o)
// 	return err
// }

func Update(ctx context.Context, o interface{}, filter interface{}, update interface{}) error {
	c := Client().Collection(CollectionName(o))
	after := options.After
	upsert := true

	// TODO filter out nil fields

	res := c.FindOneAndUpdate(ctx, filter, map[string]interface{}{"$set": update}, &options.FindOneAndUpdateOptions{
		ReturnDocument: &after,
		Upsert:         &upsert,
	})

	if err := res.Err(); err != nil {
		return err
	}

	return res.Decode(o)
}

func CollectionName(o interface{}) string {
	name := ToSnakeCase(reflect.TypeOf(o).Elem().Name())
	return pluralize.NewClient().Plural(name)
}

// ---

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

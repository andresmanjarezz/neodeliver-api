package graphql

import (
	"reflect"
	"strings"
)

func ArgToBson(p interface{}, i interface{}) map[string]interface{} {
	px, ok := p.(map[string]interface{})
	if !ok {
		return nil
	}

	res := map[string]interface{}{}

	// Find all fields provided by the graphql query & convert them to bson named within new map
	t := reflect.TypeOf(i)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		graphqlName := ReflectName(f)
		val, ok := px[graphqlName]
		if !ok {
			continue
		}

		bson := strings.Split(f.Tag.Get("bson"), ",")[0]
		if bson == "-" {
			continue
		} else if bson == "" {
			bson = strings.ToLower(f.Name)
		}

		res[bson] = val
	}

	return res
}

func ReflectName(arg reflect.StructField) string {
	name := ToSnakeCase(arg.Name)
	if n := arg.Tag.Get("graphql"); n != "" {
		name = n
	}

	return name
}

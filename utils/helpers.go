package utils

import (
	"regexp"
	"reflect"
	"strings"

	isolang "github.com/emvi/iso-639-1"
	"go.mongodb.org/mongo-driver/bson"
)

func ValidateEmail(email *string) bool {
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	match, _ := regexp.MatchString(emailRegex, *email)
	return match
}

func ValidatePhone(phone *string) bool {
	phoneRegex := `^\+\d{1,3}-\d{1,3}-\d{1,3}-\d{1,10}$`
	match, _ := regexp.MatchString(phoneRegex, *phone)
	return match
}

func ValidateLanguageCode(language *string) bool {
	return isolang.ValidCode(*language)
}

func ValidateNotificationToken(token *string) bool {
	tokenRegex := `^ExponentPushToken\[[A-Za-z0-9]+\]$`
	match, _ := regexp.MatchString(tokenRegex, *token)
	return match
}

func ConvertQueryToBSON(query string) (bson.M, error) {
	bsonMap := bson.M{}
	err := bson.UnmarshalExtJSON([]byte(query), true, &bsonMap)
	if err != nil {
		return nil, err
	}
	return bsonMap, nil
}

func GetQueryBSONDepth(obj bson.M) int {
	maxDepth := 0

	for _, value := range obj {
		if subObj, ok := value.(bson.M); ok {
			depth := GetQueryBSONDepth(subObj)
			if depth > maxDepth {
				maxDepth = depth
			}
		}
	}

	return maxDepth + 1
}

func RemoveSpaces(str *string) {
	*str = strings.ReplaceAll(*str, " ", "")
	*str = strings.ReplaceAll(*str, "\t", "")
}

func FilterNilFields(obj interface{}) interface{} {
	objValue := reflect.ValueOf(obj)
	objType := objValue.Type()
	
	// Create a new instance of the same type as the input object
	result := reflect.New(objType).Elem()

	// Iterate over the fields of the object
	for i := 0; i < objValue.NumField(); i++ {
		fieldValue := objValue.Field(i)

		// Check if the field value is nil
		if fieldValue.IsNil() {
			continue // Skip nil fields
		}

		// Set the field value in the result object
		result.Field(i).Set(fieldValue)
	}

	return result.Interface()
}

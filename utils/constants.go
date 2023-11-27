package utils

const (
	ContactStatusActive		string = "ACTIVE"
	ContactStatusInactive	string = "INACTIVE"
)

const (
	MessageEmailInvalid				string = "The email address is not valid."
	MessagePhoneNumberInvalid		string = "The phone number is not valid."
	MessageLangCodeInvalid			string = "The language code is not valid."
	MessageNotificationTokenInvalid	string = "The notification token is not valid."
	MessageSegmentQueryInvalid		string = "The filter query for segment is not valid."

	MessageDuplicationError			string = "The ID or email address is already registered within your organization."
	MessageContactCannotFindError	string = "Cannot find the contact."

	MessageTagAssignDuplicationError	string = "This tag is already assigned to this contact."
	MessageTagNameDuplicationError	string = "This tag name is already registerd within your organization."
	MessageTagNotAssignedError		string = "This tag is not assigned to this contact."
	MessageTagCannotAssignError		string = "Cannot assign tag to this contact."
	MessageTagCannotFindError		string = "Cannot find the tag."

	MessageSegmentNameDuplicationError	string = "This segment is already registered within your organization."
	MessageSegmentCannotFindError	string = "Cannot find the segment."

	MessageDefaultError				string = "Internal error occured."
)

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
	MessageContactCannotDeleteError	string = "Cannot delete the contact."
	MessageContactCannotUpdateError	string = "Cannot update the contact."

	MessageTagAssignDuplicationError	string = "This tag is already assigned to this contact."
	MessageTagNameDuplicationError	string = "This tag name is already registerd within your organization."
	MessageTagNotAssignedError		string = "This tag is not assigned to this contact."
	MessageTagCannotAssignError		string = "Cannot assign tag to this contact."
	MessageTagCannotFindError		string = "Cannot find the tag."
	MessageTagCannotDeleteError		string = "Cannot delete the tag."

	MessageSegmentNameDuplicationError	string = "This segment name is already registered."
	MessageSegmentCannotFindError	string = "Cannot find the segment."
	MessageSegmentQueryDepthExceedError	string = "The maximum depth of the query is 4."
	MessageSegmentCannotDeleteError		string = "Cannot delete the segment."

	MessageNoUpdateError			string = "There is no data to update."
	MessageDefaultError				string = "Internal error occured."
)

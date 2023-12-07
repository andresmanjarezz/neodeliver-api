package utils

const (
	ContactStatusActive   string = "ACTIVE"
	ContactStatusInactive string = "INACTIVE"
	
	CampaignMaximumRecipientNumber	int = 50
	SegmentMaximumQueryBSONDepthNumber	int = 4
)

const (
	MessageEmailInvalid             string = "The email address is not valid."
	MessagePhoneNumberInvalid       string = "The phone number is not valid."
	MessageLangCodeInvalid          string = "The language code is not valid."
	MessageNotificationTokenInvalid string = "The notification token is not valid."
	MessageSegmentQueryInvalid      string = "The filter query for segment is not valid."

	MessageDuplicationError         string = "The ID or email address is already registered within your organization."
	MessageContactCannotFindError   string = "Cannot find the contact."
	MessageContactCannotDeleteError string = "Cannot delete the contact."
	MessageContactCannotUpdateError string = "Cannot update the contact."

	MessageTagAssignDuplicationError string = "This tag is already assigned to this contact."
	MessageTagNameDuplicationError   string = "This tag name is already registerd within your organization."
	MessageTagNotAssignedError       string = "This tag is not assigned to this contact."
	MessageTagCannotAssignError      string = "Cannot assign tag to this contact."
	MessageTagCannotFindError        string = "Cannot find the tag."
	MessageTagCannotDeleteError      string = "Cannot delete the tag."

	MessageSegmentNameDuplicationError  string = "This segment name is already registered."
	MessageSegmentCannotFindError       string = "Cannot find the segment."
	MessageSegmentQueryDepthExceedError string = "The maximum depth of the query is 4."
	MessageSegmentCannotDeleteError     string = "Cannot delete the segment."

	MessageTransactionalMessageFolderCannotDeleteError string = "Cannot delete the transactional message folder."
	MessageTransactionalMessageCannotDeleteError       string = "Cannot delete the transactional message folder."

	MessageCampaignCannotFindError   string = "Cannot find the campaign."
	MessageCampaignCannotUpdateError string = "Cannot update the campaign."
	MessageCampaignCannotDeleteError string = "Cannot delete the campaign."
	MessageCampaignCannotChangeTypeError	string = "Cannot change the type of campaign once it is created."
	MessageCampaignMustBeOneTypeError	string = "Only one campaign type can be created at a time."
	MessageCampaignNoLangProvidedError	string = "At least 1 language must be provided."
	MessageCampaignRecipientExceedLimitError	string = "Exceeded maximum limit of recipients. Maximum allowed: 50"
	MessageCampaignRecipientNotProvidedError	string = "Recipients must not be empty."
	MessageCampaignInvalidRecipientError	string = "The recipients contain an invalid one."
	MessageCampaignInvalidScheduleError		string = "The campaign schedule is not valid."
	MessageCampaignInvalidLangError			string = "The language is not valid."
	MessageCampaignDuplicateLangError			string = "Each language must be present only once"
	 
	MessageNoUpdateError string = "There is no data to update."
	MessageDefaultError  string = "Internal error occured."
)

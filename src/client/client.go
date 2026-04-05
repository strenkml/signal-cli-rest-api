package client

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/h2non/filetype"

	uuid "github.com/gofrs/uuid"
	qrcode "github.com/skip2/go-qrcode"

	ds "github.com/bbernhard/signal-cli-rest-api/datastructs"
	utils "github.com/bbernhard/signal-cli-rest-api/utils"
)

const groupPrefix = "group."

const signalCliV2GroupError = "Cannot create a V2 group as self does not have a versioned profile"

type AvatarType int

const (
	GroupAvatar AvatarType = iota + 1
	ContactAvatar
	ProfileAvatar
)

type GroupPermission int

const (
	DefaultGroupPermission GroupPermission = iota + 1
	EveryMember
	OnlyAdmins
)

type GroupLinkState int

const (
	DefaultGroupLinkState GroupLinkState = iota + 1
	Enabled
	EnabledWithApproval
	Disabled
)

func (g GroupPermission) String() string {
	switch g {
	case DefaultGroupPermission:
		return ""
	case EveryMember:
		return "every-member"
	case OnlyAdmins:
		return "only-admins"
	}
	return ""
}

func (g GroupPermission) FromString(input string) GroupPermission {
	if input == "every-member" {
		return EveryMember
	}
	if input == "only-admins" {
		return OnlyAdmins
	}
	return DefaultGroupPermission
}

func (g GroupLinkState) String() string {
	switch g {
	case DefaultGroupLinkState:
		return ""
	case Enabled:
		return "enabled"
	case EnabledWithApproval:
		return "enabled-with-approval"
	case Disabled:
		return "disabled"
	}
	return ""
}

func (g GroupLinkState) FromString(input string) GroupLinkState {
	if input == "enabled" {
		return Enabled
	}
	if input == "enabled-with-approval" {
		return EnabledWithApproval
	}
	if input == "disabled" {
		return Disabled
	}

	return DefaultGroupLinkState
}

type GroupEntry struct {
	Name            string              `json:"name"`
	Description     string              `json:"description"`
	Id              string              `json:"id"`
	InternalId      string              `json:"internal_id"`
	Members         []string            `json:"members"`
	Blocked         bool                `json:"blocked"`
	PendingInvites  []string            `json:"pending_invites"`
	PendingRequests []string            `json:"pending_requests"`
	InviteLink      string              `json:"invite_link"`
	Admins          []string            `json:"admins"`
	Permissions     ds.GroupPermissions `json:"permissions"`
}

type GroupMember struct {
	Number string `json:"number"`
	Uuid   string `json:"uuid"`
}

type GroupAdmin struct {
	Number string `json:"number"`
	Uuid   string `json:"uuid"`
}

type ExpandedGroupEntry struct {
	Name            string              `json:"name"`
	Description     string              `json:"description"`
	Id              string              `json:"id"`
	InternalId      string              `json:"internal_id"`
	Members         []GroupMember       `json:"members"`
	Blocked         bool                `json:"blocked"`
	PendingInvites  []GroupMember       `json:"pending_invites"`
	PendingRequests []GroupMember       `json:"pending_requests"`
	InviteLink      string              `json:"invite_link"`
	Admins          []GroupAdmin        `json:"admins"`
	Permissions     ds.GroupPermissions `json:"permissions"`
}

type IdentityEntry struct {
	Number       string `json:"number"`
	Status       string `json:"status"`
	Fingerprint  string `json:"fingerprint"`
	Added        string `json:"added"`
	SafetyNumber string `json:"safety_number"`
	Uuid         string `json:"uuid"`
}

type SignalCliGroupEntry struct {
	Name                  string        `json:"name"`
	Description           string        `json:"description"`
	Id                    string        `json:"id"`
	IsMember              bool          `json:"isMember"`
	IsBlocked             bool          `json:"isBlocked"`
	Members               []GroupMember `json:"members"`
	PendingMembers        []GroupMember `json:"pendingMembers"`
	RequestingMembers     []GroupMember `json:"requestingMembers"`
	GroupInviteLink       string        `json:"groupInviteLink"`
	Admins                []GroupAdmin  `json:"admins"`
	Uuid                  string        `json:"uuid"`
	PermissionEditDetails string        `json:"permissionEditDetails"`
	PermissionAddMember   string        `json:"permissionAddMember"`
	PermissionSendMessage string        `json:"permissionSendMessage"`
}

type SignalCliIdentityEntry struct {
	Number                string `json:"number"`
	Uuid                  string `json:"uuid"`
	Fingerprint           string `json:"fingerprint"`
	SafetyNumber          string `json:"safetyNumber"`
	ScannableSafetyNumber string `json:"scannableSafetyNumber"`
	TrustLevel            string `json:"trustLevel"`
	AddedTimestamp        int64  `json:"addedTimestamp"`
}

type SendResponse struct {
	Timestamp int64 `json:"timestamp"`
}

type RemoteDeleteResponse struct {
	Timestamp int64 `json:"timestamp"`
}

type About struct {
	SupportedApiVersions []string            `json:"versions"`
	BuildNr              int                 `json:"build"`
	Mode                 string              `json:"mode"`
	Version              string              `json:"version"`
	Capabilities         map[string][]string `json:"capabilities"`
}

type SearchResultEntry struct {
	Number     string `json:"number"`
	Registered bool   `json:"registered"`
}

type SetUsernameResponse struct {
	Username     string `json:"username"`
	UsernameLink string `json:"username_link"`
}

type ListInstalledStickerPacksResponse struct {
	PackId    string `json:"pack_id"`
	Url       string `json:"url"`
	Installed bool   `json:"installed"`
	Title     string `json:"title"`
	Author    string `json:"author"`
}

type ContactProfile struct {
	GivenName            string `json:"given_name"`
	FamilyName           string `json:"lastname"`
	About                string `json:"about"`
	HasAvatar            bool   `json:"has_avatar"`
	LastUpdatedTimestamp int64  `json:"last_updated_timestamp"`
}

type Nickname struct {
	Name       string `json:"name"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
}

type ListContactsResponse struct {
	Number            string         `json:"number"`
	Uuid              string         `json:"uuid"`
	Name              string         `json:"name"`
	ProfileName       string         `json:"profile_name"`
	Username          string         `json:"username"`
	Color             string         `json:"color"`
	Blocked           bool           `json:"blocked"`
	MessageExpiration string         `json:"message_expiration"`
	Note              string         `json:"note"`
	Profile           ContactProfile `json:"profile"`
	GivenName         string         `json:"given_name"`
	Nickname          Nickname       `json:"nickname"`
}

type ListDevicesResponse struct {
	Id                int64  `json:"id"`
	Name              string `json:"name"`
	LastSeenTimestamp int64  `json:"last_seen_timestamp"`
	CreationTimestamp int64  `json:"creation_timestamp"`
}

func cleanupTmpFiles(paths []string) {
	for _, path := range paths {
		os.Remove(path)
	}
}

func cleanupAttachmentEntries(attachmentEntries []AttachmentEntry, linkPreviewAttachmentEntry *AttachmentEntry) {
	for _, attachmentEntry := range attachmentEntries {
		attachmentEntry.cleanUp()
	}

	if linkPreviewAttachmentEntry != nil {
		linkPreviewAttachmentEntry.cleanUp()
	}
}

func convertInternalGroupIdToGroupId(internalId string) string {
	return groupPrefix + base64.StdEncoding.EncodeToString([]byte(internalId))
}

func signalCliGroupPermissionToRestApiGroupPermission(permission string) string {
	if permission == "EVERY_MEMBER" {
		return "every-member"
	} else if permission == "ONLY_ADMINS" {
		return "only-admins"
	}

	return ""
}

func ConvertGroupIdToInternalGroupId(id string) (string, error) {
	groupIdWithoutPrefix := strings.TrimPrefix(id, groupPrefix)
	internalGroupId, err := base64.StdEncoding.DecodeString(groupIdWithoutPrefix)
	if err != nil {
		return "", errors.New("Invalid group id")
	}

	return string(internalGroupId), err
}

func getRecipientType(s string) (ds.RecpType, error) {
	if strings.HasPrefix(s, groupPrefix) {
		s1 := strings.TrimPrefix(s, groupPrefix)
		signalCliBase64EncodedGroupId, err := base64.StdEncoding.DecodeString(s1)
		if err == nil {
			signalCliGroupId, err := base64.StdEncoding.DecodeString(string(signalCliBase64EncodedGroupId))
			if err == nil {
				if len(signalCliGroupId) == 32 {
					return ds.Group, nil
				} else {
					return ds.Group, errors.New("Invalid Signal group size (" + strconv.Itoa(len(signalCliGroupId)))
				}
			}
		} else if len(s1) <= 10 {
			return ds.Username, nil
		}
		return ds.Group, errors.New("Invalid identifier " + s)
	} else if utils.IsPhoneNumber(s) {
		return ds.Number, nil
	} else {
		_, err := uuid.FromString(s)
		if err == nil {
			return ds.Number, nil
		}
	}
	return ds.Username, nil
}

type SignalClient struct {
	signalCliConfig          string
	attachmentTmpDir         string
	avatarTmpDir             string
	jsonRpc2ClientConfig     *utils.JsonRpc2ClientConfig
	jsonRpc2ClientConfigPath string
	jsonRpc2Clients          map[string]*JsonRpc2Client
	signalCliApiConfigPath   string
	signalCliApiConfig       *utils.SignalCliApiConfig
	receiveWebhookUrl        string
}

func NewSignalClient(signalCliConfig string, attachmentTmpDir string, avatarTmpDir string,
	jsonRpc2ClientConfigPath string, signalCliApiConfigPath string, receiveWebhookUrl string) *SignalClient {
	return &SignalClient{
		signalCliConfig:          signalCliConfig,
		attachmentTmpDir:         attachmentTmpDir,
		avatarTmpDir:             avatarTmpDir,
		jsonRpc2ClientConfigPath: jsonRpc2ClientConfigPath,
		jsonRpc2Clients:          make(map[string]*JsonRpc2Client),
		signalCliApiConfigPath:   signalCliApiConfigPath,
		receiveWebhookUrl:        receiveWebhookUrl,
	}
}

func (s *SignalClient) Init(maxRetries int) error {
	s.signalCliApiConfig = utils.NewSignalCliApiConfig()
	err := s.signalCliApiConfig.Load(s.signalCliApiConfigPath)
	if err != nil {
		return err
	}

	s.jsonRpc2ClientConfig = utils.NewJsonRpc2ClientConfig()
	err = s.jsonRpc2ClientConfig.Load(s.jsonRpc2ClientConfigPath)
	if err != nil {
		return err
	}

	tcpPortsNumberMapping := s.jsonRpc2ClientConfig.GetTcpPortsForNumbers()
	for number, tcpPort := range tcpPortsNumberMapping {
		s.jsonRpc2Clients[number] = NewJsonRpc2Client(s.signalCliApiConfig, number)
		err := s.jsonRpc2Clients[number].Dial("127.0.0.1:"+strconv.FormatInt(tcpPort, 10), maxRetries)
		if err != nil {
			return err
		}

		go s.jsonRpc2Clients[number].ReceiveData(number, s.receiveWebhookUrl)
	}

	return nil
}

func validateLinkPreview(message string, linkPreview *ds.LinkPreviewType) error {
	if linkPreview != nil {
		if linkPreview.Url == "" {
			return errors.New("Please provide a valid Link Preview URL")
		}

		if !strings.HasPrefix(linkPreview.Url, "https") {
			return errors.New("Link Preview URL must start with https://..")
		}

		if linkPreview.Title == "" {
			return errors.New("Please provide a valid Link Preview Title")
		}

		if !strings.Contains(message, linkPreview.Url) {
			return errors.New("Link Preview URL is missing in the message!")
		}
	}

	return nil
}

func (s *SignalClient) send(signalCliSendRequest ds.SignalCliSendRequest) (*SendResponse, error) {
	var resp SendResponse
	var linkPreviewAttachmentEntry *AttachmentEntry = nil

	if len(signalCliSendRequest.Recipients) == 0 {
		return nil, errors.New("Please specify at least one recipient")
	}

	err := validateLinkPreview(signalCliSendRequest.Message, signalCliSendRequest.LinkPreview)
	if err != nil {
		return nil, err
	}

	signalCliTextFormatStrings := []string{}
	if signalCliSendRequest.TextMode != nil && *signalCliSendRequest.TextMode == "styled" {
		textstyleParser := utils.NewTextstyleParser(signalCliSendRequest.Message)
		signalCliSendRequest.Message, signalCliTextFormatStrings = textstyleParser.Parse()
	}

	var groupId string = ""
	if signalCliSendRequest.RecipientType == ds.Group {
		if len(signalCliSendRequest.Recipients) > 1 {
			return nil, errors.New("More than one recipient is currently not allowed")
		}

		grpId, err := base64.StdEncoding.DecodeString(signalCliSendRequest.Recipients[0])
		if err != nil {
			return nil, errors.New("Invalid group id")
		}
		groupId = string(grpId)
	}

	attachmentEntries := []AttachmentEntry{}
	for _, base64Attachment := range signalCliSendRequest.Base64Attachments {
		attachmentEntry := NewAttachmentEntry(base64Attachment, s.attachmentTmpDir)

		err := attachmentEntry.storeBase64AsTemporaryFile()
		if err != nil {
			cleanupAttachmentEntries(attachmentEntries, linkPreviewAttachmentEntry)
			return nil, err
		}

		attachmentEntries = append(attachmentEntries, *attachmentEntry)
	}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return nil, err
	}

	type Request struct {
		Recipients         []string `json:"recipient,omitempty"`
		Usernames          []string `json:"username,omitempty"`
		Message            string   `json:"message"`
		GroupId            string   `json:"group-id,omitempty"`
		Attachments        []string `json:"attachment,omitempty"`
		Sticker            string   `json:"sticker,omitempty"`
		Mentions           []string `json:"mentions,omitempty"`
		QuoteTimestamp     *int64   `json:"quote-timestamp,omitempty"`
		QuoteAuthor        *string  `json:"quote-author,omitempty"`
		QuoteMessage       *string  `json:"quote-message,omitempty"`
		QuoteMentions      []string `json:"quote-mentions,omitempty"`
		TextStyles         []string `json:"text-style,omitempty"`
		EditTimestamp      *int64   `json:"edit-timestamp,omitempty"`
		NotifySelf         bool     `json:"notify-self,omitempty"`
		PreviewUrl         *string  `json:"preview-url,omitempty"`
		PreviewTitle       *string  `json:"preview-title,omitempty"`
		PreviewImage       *string  `json:"preview-image,omitempty"`
		PreviewDescription *string  `json:"preview-description,omitempty"`
		ViewOnce           bool     `json:"view-once,omitempty"`
	}

	request := Request{Message: signalCliSendRequest.Message}
	if signalCliSendRequest.RecipientType == ds.Group {
		request.GroupId = groupId
	} else if signalCliSendRequest.RecipientType == ds.Number {
		request.Recipients = signalCliSendRequest.Recipients
	} else if signalCliSendRequest.RecipientType == ds.Username {
		request.Usernames = signalCliSendRequest.Recipients
	}
	for _, attachmentEntry := range attachmentEntries {
		request.Attachments = append(request.Attachments, attachmentEntry.toDataForSignal())
	}

	if signalCliSendRequest.NotifySelf == nil || *signalCliSendRequest.NotifySelf {
		request.NotifySelf = true
	}

	if signalCliSendRequest.ViewOnce != nil && *signalCliSendRequest.ViewOnce {
		request.ViewOnce = true
	}

	request.Sticker = signalCliSendRequest.Sticker
	if signalCliSendRequest.Mentions != nil {
		request.Mentions = make([]string, len(signalCliSendRequest.Mentions))
		for i, mention := range signalCliSendRequest.Mentions {
			request.Mentions[i] = mention.ToString()
		}
	} else {
		request.Mentions = nil
	}
	request.QuoteTimestamp = signalCliSendRequest.QuoteTimestamp
	request.QuoteAuthor = signalCliSendRequest.QuoteAuthor
	request.QuoteMessage = signalCliSendRequest.QuoteMessage
	if signalCliSendRequest.QuoteMentions != nil {
		request.QuoteMentions = make([]string, len(signalCliSendRequest.QuoteMentions))
		for i, mention := range signalCliSendRequest.QuoteMentions {
			request.QuoteMentions[i] = mention.ToString()
		}
	} else {
		request.QuoteMentions = nil
	}
	request.EditTimestamp = signalCliSendRequest.EditTimestamp

	if len(signalCliTextFormatStrings) > 0 {
		request.TextStyles = signalCliTextFormatStrings
	}

	if signalCliSendRequest.LinkPreview != nil {
		request.PreviewUrl = &signalCliSendRequest.LinkPreview.Url
		request.PreviewTitle = &signalCliSendRequest.LinkPreview.Title
		request.PreviewDescription = &signalCliSendRequest.LinkPreview.Description

		if signalCliSendRequest.LinkPreview.Base64Thumbnail != "" {
			linkPreviewAttachmentEntry = NewAttachmentEntry(signalCliSendRequest.LinkPreview.Base64Thumbnail, s.attachmentTmpDir)
			err := linkPreviewAttachmentEntry.storeBase64AsTemporaryFile()
			if err != nil {
				cleanupAttachmentEntries(attachmentEntries, linkPreviewAttachmentEntry)
				return nil, err
			}
			request.PreviewImage = &linkPreviewAttachmentEntry.FilePath
		}
	}

	rawData, err := jsonRpc2Client.getRaw("send", &signalCliSendRequest.Number, request)
	if err != nil {
		cleanupAttachmentEntries(attachmentEntries, linkPreviewAttachmentEntry)
		return nil, err
	}

	err = json.Unmarshal([]byte(rawData), &resp)
	if err != nil {
		cleanupAttachmentEntries(attachmentEntries, linkPreviewAttachmentEntry)
		if strings.Contains(err.Error(), signalCliV2GroupError) {
			return nil, errors.New("Cannot send message to group - please first update your profile.")
		}
		return nil, err
	}

	cleanupAttachmentEntries(attachmentEntries, linkPreviewAttachmentEntry)

	return &resp, nil
}

func (s *SignalClient) About() About {
	about := About{
		SupportedApiVersions: []string{"v1"},
		BuildNr:              2,
		Mode:                 "json-rpc-native",
		Version:              utils.GetEnv("BUILD_VERSION", "unset"),
		Capabilities:         map[string][]string{"v1/send": []string{"quotes", "mentions"}},
	}
	return about
}

func (s *SignalClient) RegisterNumber(number string, useVoice bool, captcha string) error {
	type Request struct {
		UseVoice bool   `json:"voice,omitempty"`
		Captcha  string `json:"captcha,omitempty"`
		Account  string `json:"account,omitempty"`
	}
	request := Request{Account: number}

	if useVoice {
		request.UseVoice = useVoice
	}

	if captcha != "" {
		request.Captcha = captcha
	}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("register", nil, request)
	return err
}

func (s *SignalClient) UnregisterNumber(number string, deleteAccount bool, deleteLocalData bool) error {
	type Request struct {
		DeleteAccount bool `json:"delete-account,omitempty"`
	}
	req := Request{}

	if deleteAccount {
		req.DeleteAccount = true
	}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("unregister", &number, req)
	if err != nil {
		return err
	}

	if deleteLocalData {
		return s.DeleteLocalAccountData(number, false)
	}
	return nil
}

func (s *SignalClient) DeleteLocalAccountData(number string, ignoreRegistered bool) error {
	type Request struct {
		IgnoreRegistered bool `json:"ignore-registered,omitempty"`
	}
	req := Request{}
	if ignoreRegistered {
		req.IgnoreRegistered = true
	}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("deleteLocalAccountData", &number, req)
	return err
}

func (s *SignalClient) VerifyRegisteredNumber(number string, token string, pin string) error {
	type Request struct {
		VerificationCode string `json:"verificationCode,omitempty"`
		Account          string `json:"account,omitempty"`
		Pin              string `json:"pin,omitempty"`
	}
	request := Request{Account: number, VerificationCode: token}

	if pin != "" {
		request.Pin = pin
	}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("verify", nil, request)
	return err
}

func (s *SignalClient) getJsonRpc2Client() (*JsonRpc2Client, error) {
	if val, ok := s.jsonRpc2Clients[utils.MULTI_ACCOUNT_NUMBER]; ok {
		return val, nil
	}
	return nil, errors.New("Number not registered with JSON-RPC")
}

func (s *SignalClient) getJsonRpc2Clients() []*JsonRpc2Client {
	jsonRpc2Clients := []*JsonRpc2Client{}
	for _, client := range s.jsonRpc2Clients {
		jsonRpc2Clients = append(jsonRpc2Clients, client)
	}
	return jsonRpc2Clients
}

func (s *SignalClient) SetOnReceiveCallback(fn func(number string, msg JsonRpc2ReceivedMessage)) {
	for _, c := range s.jsonRpc2Clients {
		c.SetOnReceiveCallback(fn)
	}
}

func (s *SignalClient) SendV2(number string, message string, recps []string, base64Attachments []string, sticker string, mentions []ds.MessageMention,
	quoteTimestamp *int64, quoteAuthor *string, quoteMessage *string, quoteMentions []ds.MessageMention, textMode *string, editTimestamp *int64, notifySelf *bool,
	linkPreview *ds.LinkPreviewType, viewOnce *bool) (*[]SendResponse, error) {
	if len(recps) == 0 {
		return nil, errors.New("Please provide at least one recipient")
	}

	if number == "" {
		return nil, errors.New("Please provide a valid number")
	}

	groups := []string{}
	numbers := []string{}
	usernames := []string{}

	for _, recipient := range recps {
		recipientType, err := getRecipientType(recipient)
		if err != nil {
			return nil, err
		}

		if recipientType == ds.Group {
			groups = append(groups, strings.TrimPrefix(recipient, groupPrefix))
		} else if recipientType == ds.Number {
			numbers = append(numbers, recipient)
		} else if recipientType == ds.Username {
			usernames = append(usernames, recipient)
		} else {
			return nil, errors.New("Invalid recipient type")
		}
	}

	if len(numbers) > 0 && len(groups) > 0 {
		return nil, errors.New("Signal Messenger Groups and phone numbers cannot be specified together in one request! Please split them up into multiple REST API calls.")
	}

	if len(usernames) > 0 && len(groups) > 0 {
		return nil, errors.New("Signal Messenger Groups and usernames cannot be specified together in one request! Please split them up into multiple REST API calls.")
	}

	if len(numbers) > 0 && len(usernames) > 0 {
		return nil, errors.New("Signal Messenger phone numbers and usernames cannot be specified together in one request! Please split them up into multiple REST API calls.")
	}

	if len(groups) > 1 {
		return nil, errors.New("A signal message cannot be sent to more than one group at once! Please use multiple REST API calls for that.")
	}

	timestamps := []SendResponse{}
	for _, group := range groups {
		signalCliSendRequest := ds.SignalCliSendRequest{Number: number, Message: message, Recipients: []string{group}, Base64Attachments: base64Attachments,
			RecipientType: ds.Group, Sticker: sticker, Mentions: mentions, QuoteTimestamp: quoteTimestamp,
			QuoteAuthor: quoteAuthor, QuoteMessage: quoteMessage, QuoteMentions: quoteMentions,
			TextMode: textMode, EditTimestamp: editTimestamp, NotifySelf: notifySelf, LinkPreview: linkPreview, ViewOnce: viewOnce}
		timestamp, err := s.send(signalCliSendRequest)
		if err != nil {
			return nil, err
		}
		timestamps = append(timestamps, *timestamp)
	}

	if len(numbers) > 0 {
		signalCliSendRequest := ds.SignalCliSendRequest{Number: number, Message: message, Recipients: numbers, Base64Attachments: base64Attachments,
			RecipientType: ds.Number, Sticker: sticker, Mentions: mentions, QuoteTimestamp: quoteTimestamp,
			QuoteAuthor: quoteAuthor, QuoteMessage: quoteMessage, QuoteMentions: quoteMentions,
			TextMode: textMode, EditTimestamp: editTimestamp, NotifySelf: notifySelf, LinkPreview: linkPreview, ViewOnce: viewOnce}
		timestamp, err := s.send(signalCliSendRequest)
		if err != nil {
			return nil, err
		}
		timestamps = append(timestamps, *timestamp)
	}

	if len(usernames) > 0 {
		signalCliSendRequest := ds.SignalCliSendRequest{Number: number, Message: message, Recipients: usernames, Base64Attachments: base64Attachments,
			RecipientType: ds.Username, Sticker: sticker, Mentions: mentions, QuoteTimestamp: quoteTimestamp,
			QuoteAuthor: quoteAuthor, QuoteMessage: quoteMessage, QuoteMentions: quoteMentions,
			TextMode: textMode, EditTimestamp: editTimestamp, NotifySelf: notifySelf, LinkPreview: linkPreview, ViewOnce: viewOnce}
		timestamp, err := s.send(signalCliSendRequest)
		if err != nil {
			return nil, err
		}
		timestamps = append(timestamps, *timestamp)
	}

	return &timestamps, nil
}

func (s *SignalClient) GetReceiveChannel() (chan JsonRpc2ReceivedMessage, string, error) {
	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return nil, "", err
	}
	return jsonRpc2Client.GetReceiveChannel()
}

func (s *SignalClient) RemoveReceiveChannel(channelUuid string) {
	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return
	}
	jsonRpc2Client.RemoveReceiveChannel(channelUuid)
}

func (s *SignalClient) CreateGroup(number string, name string, members []string, description string, editGroupPermission GroupPermission, addMembersPermission GroupPermission,
	sendMessagesPermission GroupPermission, groupLinkState GroupLinkState, expirationTime *int) (string, error) {
	type Request struct {
		Name                    string   `json:"name"`
		Members                 []string `json:"members"`
		Link                    string   `json:"link,omitempty"`
		Description             string   `json:"description,omitempty"`
		EditGroupPermissions    string   `json:"setPermissionEditDetails,omitempty"`
		AddMembersPermissions   string   `json:"setPermissionAddMember,omitempty"`
		SendMessagesPermissions string   `json:"setPermissionSendMessages,omitempty"`
		Expiration              int      `json:"expiration,omitempty"`
	}
	request := Request{Name: name, Members: prefixUsernameMembers(members)}

	if groupLinkState != DefaultGroupLinkState {
		request.Link = groupLinkState.String()
	}

	if description != "" {
		request.Description = description
	}

	if editGroupPermission != DefaultGroupPermission {
		request.EditGroupPermissions = editGroupPermission.String()
	}

	if addMembersPermission != DefaultGroupPermission {
		request.AddMembersPermissions = addMembersPermission.String()
	}

	if sendMessagesPermission != DefaultGroupPermission {
		request.SendMessagesPermissions = sendMessagesPermission.String()
	}

	if expirationTime != nil {
		request.Expiration = *expirationTime
	}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return "", err
	}
	rawData, err := jsonRpc2Client.getRaw("updateGroup", &number, request)
	if err != nil {
		return "", err
	}

	type Response struct {
		GroupId   string `json:"groupId"`
		Timestamp int64  `json:"timestamp"`
	}
	var resp Response
	json.Unmarshal([]byte(rawData), &resp)

	groupId := convertInternalGroupIdToGroupId(resp.GroupId)
	return groupId, nil
}

func prefixUsernameMembers(members []string) []string {
	res := []string{}
	for _, member := range members {
		recipientType, err := getRecipientType(member)
		if err == nil && recipientType == ds.Username {
			res = append(res, "u:"+member)
		} else {
			res = append(res, member)
		}
	}
	return res
}

func (s *SignalClient) updateGroupMembers(number string, groupId string, members []string, add bool) error {
	if len(members) == 0 {
		return nil
	}

	group, err := s.GetGroup(number, groupId)
	if err != nil {
		return err
	}

	if group == nil {
		return &NotFoundError{Description: "No group with that group id (" + groupId + ") found"}
	}

	internalGroupId, err := ConvertGroupIdToInternalGroupId(groupId)
	if err != nil {
		return errors.New("Invalid group id")
	}

	type Request struct {
		Name          string   `json:"name,omitempty"`
		Members       []string `json:"member,omitempty"`
		RemoveMembers []string `json:"remove-member,omitempty"`
		GroupId       string   `json:"groupId"`
	}
	request := Request{GroupId: internalGroupId}
	if add {
		request.Members = append(request.Members, prefixUsernameMembers(members)...)
	} else {
		request.RemoveMembers = append(request.RemoveMembers, prefixUsernameMembers(members)...)
	}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("updateGroup", &number, request)
	return err
}

func (s *SignalClient) AddMembersToGroup(number string, groupId string, members []string) error {
	return s.updateGroupMembers(number, groupId, members, true)
}

func (s *SignalClient) RemoveMembersFromGroup(number string, groupId string, members []string) error {
	return s.updateGroupMembers(number, groupId, members, false)
}

func (s *SignalClient) updateGroupAdmins(number string, groupId string, admins []string, add bool) error {
	if len(admins) == 0 {
		return nil
	}

	group, err := s.GetGroup(number, groupId)
	if err != nil {
		return err
	}

	if group == nil {
		return &NotFoundError{Description: "No group with that group id (" + groupId + ") found"}
	}

	internalGroupId, err := ConvertGroupIdToInternalGroupId(groupId)
	if err != nil {
		return errors.New("Invalid group id")
	}

	type Request struct {
		Name         string   `json:"name,omitempty"`
		Admins       []string `json:"admin,omitempty"`
		RemoveAdmins []string `json:"remove-admin,omitempty"`
		GroupId      string   `json:"groupId"`
	}
	request := Request{GroupId: internalGroupId}
	if add {
		request.Admins = append(request.Admins, admins...)
	} else {
		request.RemoveAdmins = append(request.RemoveAdmins, admins...)
	}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("updateGroup", &number, request)
	return err
}

func (s *SignalClient) AddAdminsToGroup(number string, groupId string, admins []string) error {
	return s.updateGroupAdmins(number, groupId, admins, true)
}

func (s *SignalClient) RemoveAdminsFromGroup(number string, groupId string, admins []string) error {
	return s.updateGroupAdmins(number, groupId, admins, false)
}

func (s *SignalClient) GetGroupsExpanded(number string) ([]ExpandedGroupEntry, error) {
	groupEntries := []ExpandedGroupEntry{}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return groupEntries, err
	}
	rawData, err := jsonRpc2Client.getRaw("listGroups", &number, nil)
	if err != nil {
		return groupEntries, err
	}

	var signalCliGroupEntries []SignalCliGroupEntry
	err = json.Unmarshal([]byte(rawData), &signalCliGroupEntries)
	if err != nil {
		return groupEntries, err
	}

	for _, signalCliGroupEntry := range signalCliGroupEntries {
		var groupEntry ExpandedGroupEntry
		groupEntry.InternalId = signalCliGroupEntry.Id
		groupEntry.Name = signalCliGroupEntry.Name
		groupEntry.Id = convertInternalGroupIdToGroupId(signalCliGroupEntry.Id)
		groupEntry.Blocked = signalCliGroupEntry.IsBlocked
		groupEntry.Description = signalCliGroupEntry.Description
		groupEntry.Permissions.SendMessages = signalCliGroupPermissionToRestApiGroupPermission(signalCliGroupEntry.PermissionSendMessage)
		groupEntry.Permissions.EditGroup = signalCliGroupPermissionToRestApiGroupPermission(signalCliGroupEntry.PermissionSendMessage)
		groupEntry.Permissions.AddMembers = signalCliGroupPermissionToRestApiGroupPermission(signalCliGroupEntry.PermissionAddMember)
		groupEntry.Members = signalCliGroupEntry.Members
		groupEntry.PendingInvites = signalCliGroupEntry.PendingMembers
		groupEntry.PendingRequests = signalCliGroupEntry.RequestingMembers
		groupEntry.Admins = signalCliGroupEntry.Admins
		groupEntry.InviteLink = signalCliGroupEntry.GroupInviteLink

		groupEntries = append(groupEntries, groupEntry)
	}

	return groupEntries, nil
}

func (s *SignalClient) GetGroups(number string) ([]GroupEntry, error) {
	expandedGroupEntries, err := s.GetGroupsExpanded(number)
	if err != nil {
		return []GroupEntry{}, err
	}

	groupEntries := []GroupEntry{}
	for _, expandedGroupEntry := range expandedGroupEntries {
		groupEntry := GroupEntry{InternalId: expandedGroupEntry.InternalId, Name: expandedGroupEntry.Name,
			Id: expandedGroupEntry.Id, Blocked: expandedGroupEntry.Blocked, Description: expandedGroupEntry.Description,
			Permissions: expandedGroupEntry.Permissions, InviteLink: expandedGroupEntry.InviteLink}

		members := []string{}
		for _, val := range expandedGroupEntry.Members {
			identifier := val.Number
			if identifier == "" {
				identifier = val.Uuid
			}
			members = append(members, identifier)
		}
		groupEntry.Members = members

		pendingInvites := []string{}
		for _, val := range expandedGroupEntry.PendingInvites {
			identifier := val.Number
			if identifier == "" {
				identifier = val.Uuid
			}
			pendingInvites = append(pendingInvites, identifier)
		}
		groupEntry.PendingInvites = pendingInvites

		pendingRequests := []string{}
		for _, val := range expandedGroupEntry.PendingRequests {
			identifier := val.Number
			if identifier == "" {
				identifier = val.Uuid
			}
			pendingRequests = append(pendingRequests, identifier)
		}
		groupEntry.PendingRequests = pendingRequests

		admins := []string{}
		for _, val := range expandedGroupEntry.Admins {
			identifier := val.Number
			if identifier == "" {
				identifier = val.Uuid
			}
			admins = append(admins, identifier)
		}
		groupEntry.Admins = admins

		groupEntries = append(groupEntries, groupEntry)
	}

	return groupEntries, nil
}

func (s *SignalClient) GetGroup(number string, groupId string) (*GroupEntry, error) {
	groupEntry := GroupEntry{}
	groups, err := s.GetGroups(number)
	if err != nil {
		return nil, err
	}

	for _, group := range groups {
		if group.Id == groupId {
			groupEntry = group
			return &groupEntry, nil
		}
	}

	return nil, nil
}

func (s *SignalClient) GetGroupExpanded(number string, groupId string) (*ExpandedGroupEntry, error) {
	groupEntry := ExpandedGroupEntry{}
	groups, err := s.GetGroupsExpanded(number)
	if err != nil {
		return nil, err
	}

	for _, group := range groups {
		if group.Id == groupId {
			groupEntry = group
			return &groupEntry, nil
		}
	}

	return nil, nil
}

func (s *SignalClient) PinMessageInGroup(number string, groupId string, targetAuthor string, timestamp int64, duration int) error {
	type Request struct {
		TargetAuthor    string `json:"target-author"`
		TargetTimestamp int64  `json:"target-timestamp"`
		PinDuration     int    `json:"pin-duration"`
		GroupId         string `json:"group-id"`
	}

	req := Request{TargetAuthor: targetAuthor, TargetTimestamp: timestamp, PinDuration: duration, GroupId: groupId}
	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("sendPinMessage", &number, req)
	return err
}

func (s *SignalClient) UnpinMessageInGroup(number string, groupId string, targetAuthor string, timestamp int64) error {
	type Request struct {
		TargetAuthor    string `json:"target-author"`
		TargetTimestamp int64  `json:"target-timestamp"`
		GroupId         string `json:"group-id"`
	}

	req := Request{TargetAuthor: targetAuthor, TargetTimestamp: timestamp, GroupId: groupId}
	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("sendUnpinMessage", &number, req)
	return err
}

func (s *SignalClient) GetAvatar(number string, id string, avatarType AvatarType) ([]byte, error) {
	var err error

	if avatarType == GroupAvatar {
		id, err = ConvertGroupIdToInternalGroupId(id)
		if err != nil {
			return []byte{}, errors.New("Invalid group id")
		}
	}

	type Request struct {
		GroupId string `json:"groupId,omitempty"`
		Contact string `json:"contact,omitempty"`
		Profile string `json:"profile,omitempty"`
	}

	var request Request

	if avatarType == GroupAvatar {
		request.GroupId = id
	} else if avatarType == ContactAvatar {
		request.Contact = id
	} else if avatarType == ProfileAvatar {
		request.Profile = id
	}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return []byte{}, err
	}
	rawData, err := jsonRpc2Client.getRaw("getAvatar", &number, request)
	if err != nil {
		if err.Error() == "Could not find avatar" {
			return []byte{}, &NotFoundError{Description: "No avatar found."}
		}
		return []byte{}, err
	}

	type SignalCliResponse struct {
		Data string `json:"data"`
	}
	var signalCliResponse SignalCliResponse
	err = json.Unmarshal([]byte(rawData), &signalCliResponse)
	if err != nil {
		return []byte{}, errors.New("Couldn't unmarshal data: " + err.Error())
	}

	groupAvatarBytes, err := base64.StdEncoding.DecodeString(signalCliResponse.Data)
	if err != nil {
		return []byte{}, errors.New("Couldn't decode base64 encoded group avatar: " + err.Error())
	}

	return groupAvatarBytes, nil
}

func (s *SignalClient) DeleteGroup(number string, groupId string) error {
	type Request struct {
		GroupId string `json:"groupId"`
	}
	request := Request{GroupId: groupId}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("quitGroup", &number, request)
	return err
}

func (s *SignalClient) GetQrCodeLink(deviceName string, qrCodeVersion int) ([]byte, error) {
	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return []byte{}, err
	}

	type StartRequest struct{}
	type Response struct {
		DeviceLinkUri string `json:"deviceLinkUri"`
	}

	result, err := jsonRpc2Client.getRaw("startLink", nil, &StartRequest{})
	if err != nil {
		return []byte{}, errors.New("Couldn't create QR code: " + err.Error())
	}

	var resp Response
	err = json.Unmarshal([]byte(result), &resp)
	if err != nil {
		return []byte{}, errors.New("Couldn't create QR code: " + err.Error())
	}

	q, err := qrcode.NewWithForcedVersion(string(resp.DeviceLinkUri), qrCodeVersion, qrcode.Highest)
	if err != nil {
		return []byte{}, errors.New("Couldn't create QR code: " + err.Error())
	}

	var png []byte
	png, err = q.PNG(256)
	if err != nil {
		return []byte{}, errors.New("Couldn't create QR code: " + err.Error())
	}

	s.finishLinkAsync(jsonRpc2Client, deviceName, resp.DeviceLinkUri)

	return png, nil
}

func (s *SignalClient) GetDeviceLinkUri(deviceName string) (string, error) {
	type StartResponse struct {
		DeviceLinkUri string `json:"deviceLinkUri"`
	}
	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return "", err
	}

	raw, err := jsonRpc2Client.getRaw("startLink", nil, struct{}{})
	if err != nil {
		return "", errors.New("Couldn't start link: " + err.Error())
	}

	var resp StartResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return "", errors.New("Couldn't parse startLink response: " + err.Error())
	}

	s.finishLinkAsync(jsonRpc2Client, deviceName, resp.DeviceLinkUri)
	return resp.DeviceLinkUri, nil
}

func (s *SignalClient) finishLinkAsync(jsonRpc2Client *JsonRpc2Client, deviceName string, deviceLinkUri string) {
	type finishRequest struct {
		DeviceLinkUri string `json:"deviceLinkUri"`
		DeviceName    string `json:"deviceName"`
	}

	go func() {
		req := finishRequest{DeviceLinkUri: deviceLinkUri, DeviceName: deviceName}
		result, err := jsonRpc2Client.getRaw("finishLink", nil, &req)
		if err != nil {
			log.Debug("Error linking device: ", err.Error())
			return
		}
		log.Debug("Linking device result: ", result)
		s.signalCliApiConfig.Load(s.signalCliApiConfigPath)
	}()
}

func (s *SignalClient) GetAccounts() ([]string, error) {
	accounts := make([]string, 0)

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return accounts, err
	}
	rawData, err := jsonRpc2Client.getRaw("listAccounts", nil, nil)
	if err != nil {
		return accounts, err
	}

	type Account struct {
		Number string `json:"number"`
	}
	accountObjs := []Account{}

	err = json.Unmarshal([]byte(rawData), &accountObjs)
	if err != nil {
		return accounts, err
	}

	for _, account := range accountObjs {
		accounts = append(accounts, account.Number)
	}

	return accounts, nil
}

func (s *SignalClient) GetAttachments() ([]string, error) {
	files := []string{}

	attachmentsPath := s.signalCliConfig + "/attachments/"
	if _, err := os.Stat(attachmentsPath); !os.IsNotExist(err) {
		err = filepath.Walk(attachmentsPath, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			files = append(files, filepath.Base(path))
			return nil
		})
		if err != nil {
			return files, err
		}
	} else {
		return files, nil
	}

	return files, nil
}

func (s *SignalClient) RemoveAttachment(attachment string) error {
	path, err := securejoin.SecureJoin(s.signalCliConfig+"/attachments/", attachment)
	if err != nil {
		return &InvalidNameError{Description: "Please provide a valid attachment name"}
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &NotFoundError{Description: "No attachment with that name found"}
	}
	err = os.Remove(path)
	if err != nil {
		return &InternalError{Description: "Couldn't delete attachment - please try again later"}
	}

	return nil
}

func (s *SignalClient) GetAttachment(attachment string) ([]byte, error) {
	path, err := securejoin.SecureJoin(s.signalCliConfig+"/attachments/", attachment)
	if err != nil {
		return []byte{}, &InvalidNameError{Description: "Please provide a valid attachment name"}
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []byte{}, &NotFoundError{Description: "No attachment with that name found"}
	}

	attachmentBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return []byte{}, &InternalError{Description: "Couldn't read attachment - please try again later"}
	}

	return attachmentBytes, nil
}

func (s *SignalClient) UpdateProfile(number string, profileName string, base64Avatar string, about *string) error {
	var err error
	var avatarTmpPath string
	if base64Avatar != "" {
		u, err := uuid.NewV4()
		if err != nil {
			return err
		}

		avatarBytes, err := base64.StdEncoding.DecodeString(base64Avatar)
		if err != nil {
			return errors.New("Couldn't decode base64 encoded avatar: " + err.Error())
		}

		fType, err := filetype.Get(avatarBytes)
		if err != nil {
			return err
		}

		avatarTmpPath = s.avatarTmpDir + u.String() + "." + fType.Extension

		f, err := os.Create(avatarTmpPath)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := f.Write(avatarBytes); err != nil {
			cleanupTmpFiles([]string{avatarTmpPath})
			return err
		}
		if err := f.Sync(); err != nil {
			cleanupTmpFiles([]string{avatarTmpPath})
			return err
		}
		f.Close()
	}

	type Request struct {
		Name         string  `json:"given-name"`
		Avatar       string  `json:"avatar,omitempty"`
		RemoveAvatar bool    `json:"remove-avatar"`
		About        *string `json:"about,omitempty"`
	}
	request := Request{Name: profileName}
	request.About = about
	if base64Avatar == "" {
		request.RemoveAvatar = true
	} else {
		request.Avatar = avatarTmpPath
		request.RemoveAvatar = false
	}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("updateProfile", &number, request)

	cleanupTmpFiles([]string{avatarTmpPath})
	return err
}

func (s *SignalClient) ListIdentities(number string) (*[]IdentityEntry, error) {
	identityEntries := []IdentityEntry{}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return nil, err
	}
	rawData, err := jsonRpc2Client.getRaw("listIdentities", &number, nil)
	if err != nil {
		return nil, err
	}

	signalCliIdentityEntries := []SignalCliIdentityEntry{}
	err = json.Unmarshal([]byte(rawData), &signalCliIdentityEntries)
	if err != nil {
		return nil, err
	}
	for _, signalCliIdentityEntry := range signalCliIdentityEntries {
		identityEntry := IdentityEntry{
			Number:       signalCliIdentityEntry.Number,
			Status:       signalCliIdentityEntry.TrustLevel,
			Added:        strconv.FormatInt(signalCliIdentityEntry.AddedTimestamp, 10),
			Fingerprint:  signalCliIdentityEntry.Fingerprint,
			SafetyNumber: signalCliIdentityEntry.SafetyNumber,
			Uuid:         signalCliIdentityEntry.Uuid,
		}
		identityEntries = append(identityEntries, identityEntry)
	}

	return &identityEntries, nil
}

func (s *SignalClient) TrustIdentity(number string, numberToTrust string, verifiedSafetyNumber *string, trustAllKnownKeys *bool) error {
	type Request struct {
		VerifiedSafetyNumber string `json:"verified-safety-number,omitempty"`
		TrustAllKnownKeys    bool   `json:"trust-all-known-keys,omitempty"`
		Recipient            string `json:"recipient"`
	}
	request := Request{Recipient: numberToTrust}

	if verifiedSafetyNumber != nil {
		request.VerifiedSafetyNumber = *verifiedSafetyNumber
	}

	if trustAllKnownKeys != nil {
		request.TrustAllKnownKeys = *trustAllKnownKeys
	}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("trust", &number, request)
	return err
}

func (s *SignalClient) BlockGroup(number string, groupId string) error {
	type Request struct {
		GroupId string `json:"groupId"`
	}
	request := Request{GroupId: groupId}
	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("block", &number, request)
	return err
}

func (s *SignalClient) JoinGroup(number string, groupId string) error {
	type Request struct {
		GroupId string `json:"groupId"`
	}
	request := Request{GroupId: groupId}
	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("updateGroup", &number, request)
	return err
}

func (s *SignalClient) QuitGroup(number string, groupId string) error {
	type Request struct {
		GroupId string `json:"groupId"`
	}
	request := Request{GroupId: groupId}
	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("quitGroup", &number, request)
	return err
}

func (s *SignalClient) UpdateGroup(number string, groupId string, base64Avatar *string, groupDescription *string, groupName *string, expirationTime *int,
	groupLinkState *GroupLinkState, editGroupPermission GroupPermission, addMembersPermission GroupPermission, sendMessagesPermission GroupPermission) error {
	var err error
	var avatarTmpPath string = ""
	if base64Avatar != nil {
		u, err := uuid.NewV4()
		if err != nil {
			return err
		}

		avatarBytes, err := base64.StdEncoding.DecodeString(*base64Avatar)
		if err != nil {
			return errors.New("Couldn't decode base64 encoded avatar: " + err.Error())
		}

		fType, err := filetype.Get(avatarBytes)
		if err != nil {
			return err
		}

		avatarTmpPath = s.avatarTmpDir + u.String() + "." + fType.Extension

		f, err := os.Create(avatarTmpPath)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := f.Write(avatarBytes); err != nil {
			cleanupTmpFiles([]string{avatarTmpPath})
			return err
		}
		if err := f.Sync(); err != nil {
			cleanupTmpFiles([]string{avatarTmpPath})
			return err
		}
		f.Close()
	}

	type Request struct {
		GroupId                 string  `json:"groupId"`
		Avatar                  string  `json:"avatar,omitempty"`
		Description             *string `json:"description,omitempty"`
		Name                    *string `json:"name,omitempty"`
		Expiration              int     `json:"expiration,omitempty"`
		Link                    string  `json:"link,omitempty"`
		EditGroupPermissions    string  `json:"setPermissionEditDetails,omitempty"`
		AddMembersPermissions   string  `json:"setPermissionAddMember,omitempty"`
		SendMessagesPermissions string  `json:"setPermissionSendMessages,omitempty"`
	}
	request := Request{GroupId: groupId}

	if base64Avatar != nil {
		request.Avatar = avatarTmpPath
	}

	request.Description = groupDescription
	request.Name = groupName

	if expirationTime != nil {
		request.Expiration = *expirationTime
	}

	if groupLinkState != nil {
		request.Link = (*groupLinkState).String()
	}

	if editGroupPermission != DefaultGroupPermission {
		request.EditGroupPermissions = editGroupPermission.String()
	}

	if addMembersPermission != DefaultGroupPermission {
		request.AddMembersPermissions = addMembersPermission.String()
	}

	if sendMessagesPermission != DefaultGroupPermission {
		request.SendMessagesPermissions = sendMessagesPermission.String()
	}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("updateGroup", &number, request)

	if avatarTmpPath != "" {
		cleanupTmpFiles([]string{avatarTmpPath})
	}

	return err
}

func (s *SignalClient) SendReaction(number string, recipient string, emoji string, target_author string, timestamp int64, remove bool) error {
	var err error
	recp := recipient
	isGroup := false
	if strings.HasPrefix(recipient, groupPrefix) {
		isGroup = true
		recp, err = ConvertGroupIdToInternalGroupId(recipient)
		if err != nil {
			return errors.New("Invalid group id")
		}
	}
	if remove && emoji == "" {
		emoji = "👍"
	}

	type Request struct {
		Recipient    string `json:"recipient,omitempty"`
		GroupId      string `json:"group-id,omitempty"`
		Emoji        string `json:"emoji"`
		TargetAuthor string `json:"target-author"`
		Timestamp    int64  `json:"target-timestamp"`
		Remove       bool   `json:"remove,omitempty"`
	}
	request := Request{}
	if !isGroup {
		request.Recipient = recp
	} else {
		request.GroupId = recp
	}
	request.Emoji = emoji
	request.TargetAuthor = target_author
	request.Timestamp = timestamp
	if remove {
		request.Remove = remove
	}
	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("sendReaction", &number, request)
	return err
}

func (s *SignalClient) SendReceipt(number string, recipient string, receipt_type string, timestamp int64) error {
	type Request struct {
		Recipient   string `json:"recipient,omitempty"`
		ReceiptType string `json:"receipt-type"`
		Timestamp   int64  `json:"target-timestamp"`
	}
	request := Request{Recipient: recipient, ReceiptType: receipt_type, Timestamp: timestamp}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("sendReceipt", &number, request)
	return err
}

func (s *SignalClient) SendStartTyping(number string, recipient string) error {
	var err error
	recp := recipient
	isGroup := false
	if strings.HasPrefix(recipient, groupPrefix) {
		isGroup = true
		recp, err = ConvertGroupIdToInternalGroupId(recipient)
		if err != nil {
			return errors.New("Invalid group id")
		}
	}

	type Request struct {
		Recipient string `json:"recipient,omitempty"`
		GroupId   string `json:"group-id,omitempty"`
	}
	request := Request{}
	if !isGroup {
		request.Recipient = recp
	} else {
		request.GroupId = recp
	}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("sendTyping", &number, request)
	return err
}

func (s *SignalClient) SendStopTyping(number string, recipient string) error {
	var err error
	recp := recipient
	isGroup := false
	if strings.HasPrefix(recipient, groupPrefix) {
		isGroup = true
		recp, err = ConvertGroupIdToInternalGroupId(recipient)
		if err != nil {
			return errors.New("Invalid group id")
		}
	}

	type Request struct {
		Recipient string `json:"recipient,omitempty"`
		GroupId   string `json:"group-id,omitempty"`
		Stop      bool   `json:"stop"`
	}
	request := Request{Stop: true}
	if !isGroup {
		request.Recipient = recp
	} else {
		request.GroupId = recp
	}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("sendTyping", &number, request)
	return err
}

func (s *SignalClient) SearchForNumbers(number string, numbers []string) ([]SearchResultEntry, error) {
	searchResultEntries := []SearchResultEntry{}

	type Request struct {
		Numbers []string `json:"recipient"`
	}
	request := Request{Numbers: numbers}

	jsonRpc2Clients := s.getJsonRpc2Clients()
	if len(jsonRpc2Clients) == 0 {
		return searchResultEntries, errors.New("No JsonRpc2Client registered!")
	}

	var err error
	var rawData string
	for _, jsonRpc2Client := range jsonRpc2Clients {
		rawData, err = jsonRpc2Client.getRaw("getUserStatus", &number, request)
		if err == nil {
			break
		}
	}

	if err != nil {
		return searchResultEntries, err
	}

	type SignalCliResponse struct {
		Number       string `json:"number"`
		IsRegistered bool   `json:"isRegistered"`
	}

	var resp []SignalCliResponse
	err = json.Unmarshal([]byte(rawData), &resp)
	if err != nil {
		return searchResultEntries, err
	}

	for _, val := range resp {
		searchResultEntry := SearchResultEntry{Number: val.Number, Registered: val.IsRegistered}
		searchResultEntries = append(searchResultEntries, searchResultEntry)
	}

	return searchResultEntries, err
}

func (s *SignalClient) SendContacts(number string) error {
	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("sendContacts", &number, nil)
	return err
}

func (s *SignalClient) UpdateContact(number string, recipient string, name *string, expirationInSeconds *int) error {
	type Request struct {
		Recipient  string `json:"recipient"`
		Name       string `json:"name,omitempty"`
		Expiration int    `json:"expiration,omitempty"`
	}
	request := Request{Recipient: recipient}
	if name != nil {
		request.Name = *name
	}
	if expirationInSeconds != nil {
		request.Expiration = *expirationInSeconds
	}
	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("updateContact", &number, request)
	return err
}

func (s *SignalClient) AddDevice(number string, uri string) error {
	type Request struct {
		Uri string `json:"uri"`
	}
	request := Request{Uri: uri}
	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("addDevice", &number, request)
	return err
}

func (s *SignalClient) ListDevices(number string) ([]ListDevicesResponse, error) {
	resp := []ListDevicesResponse{}

	type ListDevicesSignalCliResponse struct {
		Id                int64  `json:"id"`
		Name              string `json:"name"`
		CreatedTimestamp  int64  `json:"createdTimestamp"`
		LastSeenTimestamp int64  `json:"lastSeenTimestamp"`
	}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return resp, err
	}
	rawData, err := jsonRpc2Client.getRaw("listDevices", &number, nil)
	if err != nil {
		return resp, err
	}

	var signalCliResp []ListDevicesSignalCliResponse
	err = json.Unmarshal([]byte(rawData), &signalCliResp)
	if err != nil {
		return resp, err
	}

	for _, entry := range signalCliResp {
		deviceEntry := ListDevicesResponse{
			Id:                entry.Id,
			Name:              entry.Name,
			CreationTimestamp: entry.CreatedTimestamp,
			LastSeenTimestamp: entry.LastSeenTimestamp,
		}
		resp = append(resp, deviceEntry)
	}

	return resp, nil
}

func (s *SignalClient) RemoveDevice(number string, deviceId int64) error {
	type Request struct {
		DeviceId int64 `json:"deviceId"`
	}
	request := Request{DeviceId: deviceId}
	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("removeDevice", &number, request)
	return err
}

func (s *SignalClient) SetTrustMode(number string, trustMode utils.SignalCliTrustMode) error {
	s.signalCliApiConfig.SetTrustModeForNumber(number, trustMode)
	return s.signalCliApiConfig.Persist()
}

func (s *SignalClient) GetTrustMode(number string) utils.SignalCliTrustMode {
	trustMode, err := s.signalCliApiConfig.GetTrustModeForNumber(number)
	if err != nil {
		return utils.OnFirstUseTrust
	}
	return trustMode
}

func (s *SignalClient) SubmitRateLimitChallenge(number string, challengeToken string, captcha string) error {
	type Request struct {
		Challenge string `json:"challenge"`
		Captcha   string `json:"captcha"`
	}
	request := Request{Challenge: challengeToken, Captcha: captcha}
	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("submitRateLimitChallenge", &number, request)
	return err
}

func (s *SignalClient) SetUsername(number string, username string) (SetUsernameResponse, error) {
	type SetUsernameSignalCliResponse struct {
		Username     string `json:"username"`
		UsernameLink string `json:"usernameLink"`
	}

	var resp SetUsernameResponse

	type Request struct {
		Username string `json:"username"`
	}
	request := Request{Username: username}
	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return resp, err
	}
	rawData, err := jsonRpc2Client.getRaw("updateAccount", &number, request)

	var signalCliResp SetUsernameSignalCliResponse
	err = json.Unmarshal([]byte(rawData), &signalCliResp)
	if err != nil {
		return resp, errors.New("Couldn't process request - invalid signal-cli response")
	}

	resp.Username = signalCliResp.Username
	resp.UsernameLink = signalCliResp.UsernameLink

	return resp, err
}

func (s *SignalClient) RemoveUsername(number string) error {
	type Request struct {
		DeleteUsername bool `json:"delete-username"`
	}
	request := Request{DeleteUsername: true}
	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("updateAccount", &number, request)
	return err
}

func (s *SignalClient) UpdateAccountSettings(number string, discoverableByNumber *bool, shareNumber *bool) error {
	type Request struct {
		ShareNumber          *bool `json:"number-sharing"`
		DiscoverableByNumber *bool `json:"discoverable-by-number"`
	}
	request := Request{}
	request.DiscoverableByNumber = discoverableByNumber
	request.ShareNumber = shareNumber

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("updateAccount", &number, request)
	return err
}

func (s *SignalClient) ListInstalledStickerPacks(number string) ([]ListInstalledStickerPacksResponse, error) {
	type ListInstalledStickerPacksSignalCliResponse struct {
		PackId    string `json:"packId"`
		Url       string `json:"url"`
		Installed bool   `json:"installed"`
		Title     string `json:"title"`
		Author    string `json:"author"`
	}

	resp := []ListInstalledStickerPacksResponse{}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return resp, err
	}
	rawData, err := jsonRpc2Client.getRaw("listStickerPacks", &number, nil)
	if err != nil {
		return resp, err
	}

	var signalCliResp []ListInstalledStickerPacksSignalCliResponse
	err = json.Unmarshal([]byte(rawData), &signalCliResp)
	if err != nil {
		return resp, errors.New("Couldn't process request - invalid signal-cli response")
	}

	for _, value := range signalCliResp {
		resp = append(resp, ListInstalledStickerPacksResponse{PackId: value.PackId, Url: value.Url,
			Installed: value.Installed, Title: value.Title, Author: value.Author})
	}

	return resp, nil
}

func (s *SignalClient) AddStickerPack(number string, packId string, packKey string) error {
	stickerPackUri := fmt.Sprintf(`https://signal.art/addstickers/#pack_id=%s&pack_key=%s`, packId, packKey)

	type Request struct {
		Uri string `json:"uri"`
	}
	request := Request{Uri: stickerPackUri}
	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("addStickerPack", &number, request)
	return err
}

func (s *SignalClient) ListContacts(number string, allRecipients bool, recipient string) ([]ListContactsResponse, error) {
	type SignalCliProfileResponse struct {
		LastUpdateTimestamp int64  `json:"lastUpdateTimestamp"`
		GivenName           string `json:"givenName"`
		FamilyName          string `json:"familyName"`
		About               string `json:"about"`
		HasAvatar           bool   `json:"hasAvatar"`
	}

	type ListContactsSignalCliResponse struct {
		Number            string                   `json:"number"`
		Uuid              string                   `json:"uuid"`
		Name              string                   `json:"name"`
		ProfileName       string                   `json:"profileName"`
		Username          string                   `json:"username"`
		Color             string                   `json:"color"`
		Blocked           bool                     `json:"blocked"`
		MessageExpiration string                   `json:"messageExpiration"`
		Note              string                   `json:"note"`
		GivenName         string                   `json:"givenName"`
		Profile           SignalCliProfileResponse `json:"profile"`
		Nickname          string                   `json:"nickName"`
		NickGivenName     string                   `json:"nickGivenName"`
		NickFamilyName    string                   `json:"nickFamilyName"`
	}

	resp := []ListContactsResponse{}

	type Request struct {
		AllRecipients bool   `json:"allRecipients,omitempty"`
		Recipient     string `json:"recipient,omitempty"`
	}
	req := Request{}
	if allRecipients {
		req.AllRecipients = allRecipients
	}
	if recipient != "" {
		req.Recipient = recipient
	}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return nil, err
	}
	rawData, err := jsonRpc2Client.getRaw("listContacts", &number, req)
	if err != nil {
		return resp, err
	}

	var signalCliResp []ListContactsSignalCliResponse
	err = json.Unmarshal([]byte(rawData), &signalCliResp)
	if err != nil {
		log.Error("Couldn't list contacts", err.Error())
		return resp, errors.New("Couldn't process request - invalid signal-cli response")
	}

	if recipient != "" && len(signalCliResp) == 0 {
		return resp, &NotFoundError{Description: "No user with that id (" + recipient + ") found"}
	}

	for _, value := range signalCliResp {
		entry := ListContactsResponse{
			Number:            value.Number,
			Uuid:              value.Uuid,
			Name:              value.Name,
			ProfileName:       value.ProfileName,
			Username:          value.Username,
			Color:             value.Color,
			Blocked:           value.Blocked,
			MessageExpiration: value.MessageExpiration,
			Note:              value.Note,
			GivenName:         value.GivenName,
		}
		entry.Profile.About = value.Profile.About
		entry.Profile.HasAvatar = value.Profile.HasAvatar
		entry.Profile.LastUpdatedTimestamp = value.Profile.LastUpdateTimestamp
		entry.Profile.GivenName = value.Profile.GivenName
		entry.Profile.FamilyName = value.Profile.FamilyName
		entry.Nickname.Name = value.Nickname
		entry.Nickname.GivenName = value.NickGivenName
		entry.Nickname.FamilyName = value.NickFamilyName
		resp = append(resp, entry)
	}

	return resp, nil
}

func (s *SignalClient) SetPin(number string, registrationLockPin string) error {
	type Request struct {
		RegistrationLockPin string `json:"pin"`
	}
	req := Request{RegistrationLockPin: registrationLockPin}
	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("setPin", &number, req)
	return err
}

func (s *SignalClient) RemovePin(number string) error {
	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}
	_, err = jsonRpc2Client.getRaw("removePin", &number, nil)
	return err
}

func (s *SignalClient) RemoteDelete(number string, recipient string, timestamp int64) (RemoteDeleteResponse, error) {
	var resp RemoteDeleteResponse

	recp := recipient
	isGroup := false

	recipientType, err := getRecipientType(recipient)
	if err != nil {
		return resp, err
	}

	if recipientType == ds.Group {
		isGroup = true
		recp, err = ConvertGroupIdToInternalGroupId(recipient)
		if err != nil {
			return resp, errors.New("Invalid group id")
		}
	} else if recipientType != ds.Number && recipientType != ds.Username {
		return resp, errors.New("Invalid recipient type")
	}

	type Request struct {
		Recipient string `json:"recipient,omitempty"`
		GroupId   string `json:"group-id,omitempty"`
		Timestamp int64  `json:"target-timestamp"`
	}
	request := Request{}
	if !isGroup {
		request.Recipient = recp
	} else {
		request.GroupId = recp
	}
	request.Timestamp = timestamp

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return resp, err
	}
	rawData, err := jsonRpc2Client.getRaw("remoteDelete", &number, request)
	if err != nil {
		return resp, err
	}

	err = json.Unmarshal([]byte(rawData), &resp)
	if err != nil {
		return resp, errors.New("Couldn't process request - invalid signal-cli response")
	}

	return resp, err
}

func (s *SignalClient) CreatePoll(number string, recipient string, question string, answers []string, allowMultipleSelections bool) (string, error) {
	var rawData string

	type Response struct {
		Timestamp int64 `json:"timestamp"`
	}

	recp := recipient
	recipientType, err := getRecipientType(recipient)
	if err != nil {
		return "", err
	}

	if recipientType == ds.Group {
		recp, err = ConvertGroupIdToInternalGroupId(recipient)
		if err != nil {
			return "", errors.New("Invalid group id")
		}
	} else if recipientType != ds.Number && recipientType != ds.Username {
		return "", errors.New("Invalid recipient type")
	}

	type Request struct {
		Recipient string   `json:"recipient,omitempty"`
		GroupId   string   `json:"group-id,omitempty"`
		Username  string   `json:"username,omitempty"`
		Question  string   `json:"question"`
		Option    []string `json:"option"`
		NoMulti   bool     `json:"no-multi"`
	}

	req := Request{Question: question, Option: answers, NoMulti: !allowMultipleSelections}

	if recipientType == ds.Number {
		req.Recipient = recp
	} else if recipientType == ds.Group {
		req.GroupId = recp
	} else if recipientType == ds.Username {
		req.Username = recp
	}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return "", err
	}

	rawData, err = jsonRpc2Client.getRaw("sendPollCreate", &number, req)
	if err != nil {
		return "", err
	}

	var resp Response
	err = json.Unmarshal([]byte(rawData), &resp)
	if err != nil {
		return "", errors.New("Couldn't process request - invalid signal-cli response")
	}

	return strconv.FormatInt(resp.Timestamp, 10), nil
}

func (s *SignalClient) VoteInPoll(number string, recipient string, pollAuthor string, pollTimestamp int64, selectedAnswers []int32) error {
	recp := recipient
	recipientType, err := getRecipientType(recipient)
	if err != nil {
		return err
	}

	if recipientType == ds.Group {
		recp, err = ConvertGroupIdToInternalGroupId(recipient)
		if err != nil {
			return errors.New("Invalid group id")
		}
	} else if recipientType != ds.Number && recipientType != ds.Username {
		return errors.New("Invalid recipient type")
	}

	signalCliSelectedAnswers := []int32{}
	for _, selectedAnswer := range selectedAnswers {
		signalCliSelectedAnswers = append(signalCliSelectedAnswers, selectedAnswer-1)
	}

	type Request struct {
		Recipient       string  `json:"recipient,omitempty"`
		GroupId         string  `json:"group-id,omitempty"`
		Username        string  `json:"username,omitempty"`
		PollAuthor      string  `json:"poll-author"`
		PollTimestamp   int64   `json:"poll-timestamp"`
		SelectedAnswers []int32 `json:"option"`
		VoteCount       int32   `json:"vote-count"`
	}
	req := Request{PollAuthor: pollAuthor, PollTimestamp: pollTimestamp, SelectedAnswers: signalCliSelectedAnswers, VoteCount: 1}

	if recipientType == ds.Number {
		req.Recipient = recp
	} else if recipientType == ds.Group {
		req.GroupId = recp
	} else if recipientType == ds.Username {
		req.Username = recp
	}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}

	_, err = jsonRpc2Client.getRaw("sendPollVote", &number, req)
	return err
}

func (s *SignalClient) ClosePoll(number string, recipient string, pollTimestamp int64) error {
	recp := recipient
	recipientType, err := getRecipientType(recipient)
	if err != nil {
		return err
	}

	if recipientType == ds.Group {
		recp, err = ConvertGroupIdToInternalGroupId(recipient)
		if err != nil {
			return errors.New("Invalid group id")
		}
	} else if recipientType != ds.Number && recipientType != ds.Username {
		return errors.New("Invalid recipient type")
	}

	type Request struct {
		Recipient     string `json:"recipient,omitempty"`
		GroupId       string `json:"group-id,omitempty"`
		Username      string `json:"username,omitempty"`
		PollTimestamp int64  `json:"poll-timestamp"`
	}
	req := Request{PollTimestamp: pollTimestamp}

	if recipientType == ds.Number {
		req.Recipient = recp
	} else if recipientType == ds.Group {
		req.GroupId = recp
	} else if recipientType == ds.Username {
		req.Username = recp
	}

	jsonRpc2Client, err := s.getJsonRpc2Client()
	if err != nil {
		return err
	}

	_, err = jsonRpc2Client.getRaw("sendPollTerminate", &number, req)
	return err
}

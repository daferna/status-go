package chat

import (
	"context"
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	v1protocol "github.com/status-im/status-go/protocol/v1"
)

var (
	ErrChatNotFound            = errors.New("can't find chat")
	ErrCommunityNotFound       = errors.New("can't find community")
	ErrCommunitiesNotSupported = errors.New("communities are not supported")
	ErrChatTypeNotSupported    = errors.New("chat type not supported")
)

type ChannelGroupType string

const Personal ChannelGroupType = "personal"
const Community ChannelGroupType = "community"

type PinnedMessages struct {
	Cursor         string
	PinnedMessages []*common.PinnedMessage
}

type Member struct {
	// Community Roles
	Roles []protobuf.CommunityMember_Roles `json:"roles,omitempty"`
	// Admin indicates if the member is an admin of the group chat
	Admin bool `json:"admin"`
	// Joined indicates if the member has joined the group chat
	Joined bool `json:"joined"`
}

type Chat struct {
	ID                       string                             `json:"id"`
	Name                     string                             `json:"name"`
	Description              string                             `json:"description"`
	Color                    string                             `json:"color"`
	Emoji                    string                             `json:"emoji"`
	Active                   bool                               `json:"active"`
	ChatType                 protocol.ChatType                  `json:"chatType"`
	Timestamp                int64                              `json:"timestamp"`
	LastClockValue           uint64                             `json:"lastClockValue"`
	DeletedAtClockValue      uint64                             `json:"deletedAtClockValue"`
	ReadMessagesAtClockValue uint64                             `json:"readMessagesAtClockValue"`
	UnviewedMessagesCount    uint                               `json:"unviewedMessagesCount"`
	UnviewedMentionsCount    uint                               `json:"unviewedMentionsCount"`
	LastMessage              *common.Message                    `json:"lastMessage"`
	Members                  map[string]Member                  `json:"members,omitempty"`
	MembershipUpdates        []v1protocol.MembershipUpdateEvent `json:"membershipUpdateEvents"`
	Alias                    string                             `json:"alias,omitempty"`
	Identicon                string                             `json:"identicon"`
	Muted                    bool                               `json:"muted"`
	InvitationAdmin          string                             `json:"invitationAdmin,omitempty"`
	ReceivedInvitationAdmin  string                             `json:"receivedInvitationAdmin,omitempty"`
	Profile                  string                             `json:"profile,omitempty"`
	CommunityID              string                             `json:"communityId"`
	CategoryID               string                             `json:"categoryId"`
	Position                 int32                              `json:"position,omitempty"`
	Permissions              *protobuf.CommunityPermissions     `json:"permissions,omitempty"`
	Joined                   int64                              `json:"joined,omitempty"`
	SyncedTo                 uint32                             `json:"syncedTo,omitempty"`
	SyncedFrom               uint32                             `json:"syncedFrom,omitempty"`
	FirstMessageTimestamp    uint32                             `json:"firstMessageTimestamp,omitempty"`
	Highlight                bool                               `json:"highlight,omitempty"`
	PinnedMessages           *PinnedMessages                    `json:"pinnedMessages,omitempty"`
	CanPost                  bool                               `json:"canPost"`
	Base64Image              string                             `json:"image,omitempty"`
}

type ChannelGroup struct {
	Type                    ChannelGroupType                         `json:"channelGroupType"`
	Name                    string                                   `json:"name"`
	Images                  map[string]images.IdentityImage          `json:"images"`
	Color                   string                                   `json:"color"`
	Chats                   map[string]*Chat                         `json:"chats"`
	Categories              map[string]communities.CommunityCategory `json:"categories"`
	EnsName                 string                                   `json:"ensName"`
	Admin                   bool                                     `json:"admin"`
	Verified                bool                                     `json:"verified"`
	Description             string                                   `json:"description"`
	IntroMessage            string                                   `json:"introMessage"`
	OutroMessage            string                                   `json:"outroMessage"`
	Tags                    []communities.CommunityTag               `json:"tags"`
	Permissions             *protobuf.CommunityPermissions           `json:"permissions"`
	Members                 map[string]*protobuf.CommunityMember     `json:"members"`
	CanManageUsers          bool                                     `json:"canManageUsers"`
	Muted                   bool                                     `json:"muted"`
	BanList                 []string                                 `json:"banList"`
	Encrypted               bool                                     `json:"encrypted"`
	CommunityTokensMetadata []*protobuf.CommunityTokenMetadata       `json:"communityTokensMetadata"`
	UnviewedMessagesCount   int                                      `json:"unviewedMessagesCount"`
	UnviewedMentionsCount   int                                      `json:"unviewedMentionsCount"`
}

func NewAPI(service *Service) *API {
	return &API{
		s: service,
	}
}

type API struct {
	s *Service
}

func unique(communities []*communities.Community) (result []*communities.Community) {
	inResult := make(map[string]bool)
	for _, community := range communities {
		if _, ok := inResult[community.IDString()]; !ok {
			inResult[community.IDString()] = true
			result = append(result, community)
		}
	}
	return result
}

func (api *API) GetChannelGroups(ctx context.Context) (map[string]ChannelGroup, error) {
	joinedCommunities, err := api.s.messenger.JoinedCommunities()
	if err != nil {
		return nil, err
	}
	spectatedCommunities, err := api.s.messenger.SpectatedCommunities()
	if err != nil {
		return nil, err
	}

	pubKey := types.EncodeHex(crypto.FromECDSAPub(api.s.messenger.IdentityPublicKey()))

	result := make(map[string]ChannelGroup)

	// Get chats from cache to get unviewed	messages counts
	channels := api.s.messenger.Chats()
	totalUnviewedMessageCount := 0
	totalUnviewedMentionsCount := 0

	for _, chat := range channels {
		if !chat.IsActivePersonalChat() {
			continue
		}

		totalUnviewedMessageCount += int(chat.UnviewedMessagesCount)
		totalUnviewedMentionsCount += int(chat.UnviewedMentionsCount)
	}

	result[pubKey] = ChannelGroup{
		Type:                    Personal,
		Name:                    "",
		Images:                  make(map[string]images.IdentityImage),
		Color:                   "",
		Chats:                   make(map[string]*Chat),
		Categories:              make(map[string]communities.CommunityCategory),
		EnsName:                 "", // Not implemented yet in communities
		Admin:                   true,
		Verified:                true,
		Description:             "",
		IntroMessage:            "",
		OutroMessage:            "",
		Tags:                    []communities.CommunityTag{},
		Permissions:             &protobuf.CommunityPermissions{},
		Muted:                   false,
		CommunityTokensMetadata: []*protobuf.CommunityTokenMetadata{},
		UnviewedMessagesCount:   totalUnviewedMessageCount,
		UnviewedMentionsCount:   totalUnviewedMentionsCount,
	}

	for _, community := range unique(append(joinedCommunities, spectatedCommunities...)) {
		totalUnviewedMessageCount = 0
		totalUnviewedMentionsCount = 0

		for _, chat := range channels {
			if chat.CommunityID != community.IDString() || !chat.Active {
				continue
			}

			totalUnviewedMessageCount += int(chat.UnviewedMessagesCount)
			totalUnviewedMentionsCount += int(chat.UnviewedMentionsCount)
		}

		chGrp := ChannelGroup{
			Type:                    Community,
			Name:                    community.Name(),
			Color:                   community.Color(),
			Images:                  make(map[string]images.IdentityImage),
			Chats:                   make(map[string]*Chat),
			Categories:              make(map[string]communities.CommunityCategory),
			Admin:                   community.IsAdmin(),
			Verified:                community.Verified(),
			Description:             community.DescriptionText(),
			IntroMessage:            community.IntroMessage(),
			OutroMessage:            community.OutroMessage(),
			Tags:                    community.Tags(),
			Permissions:             community.Description().Permissions,
			Members:                 community.Description().Members,
			CanManageUsers:          community.CanManageUsers(community.MemberIdentity()),
			Muted:                   community.Muted(),
			BanList:                 community.Description().BanList,
			Encrypted:               community.Encrypted(),
			CommunityTokensMetadata: community.Description().CommunityTokensMetadata,
			UnviewedMessagesCount:   totalUnviewedMessageCount,
			UnviewedMentionsCount:   totalUnviewedMentionsCount,
		}

		for t, i := range community.Images() {
			chGrp.Images[t] = images.IdentityImage{Name: t, Payload: i.Payload}
		}

		result[community.IDString()] = chGrp
	}

	return result, nil
}

func (api *API) GetChatsByChannelGroupID(ctx context.Context, channelGroupID string) (*ChannelGroup, error) {
	pubKey := types.EncodeHex(crypto.FromECDSAPub(api.s.messenger.IdentityPublicKey()))

	if pubKey == channelGroupID {
		result := &ChannelGroup{
			Type:                    Personal,
			Name:                    "",
			Images:                  make(map[string]images.IdentityImage),
			Color:                   "",
			Chats:                   make(map[string]*Chat),
			Categories:              make(map[string]communities.CommunityCategory),
			EnsName:                 "", // Not implemented yet in communities
			Admin:                   true,
			Verified:                true,
			Description:             "",
			IntroMessage:            "",
			OutroMessage:            "",
			Tags:                    []communities.CommunityTag{},
			Permissions:             &protobuf.CommunityPermissions{},
			Muted:                   false,
			CommunityTokensMetadata: []*protobuf.CommunityTokenMetadata{},
		}

		channels := api.s.messenger.Chats()

		for _, chat := range channels {
			if !chat.IsActivePersonalChat() {
				continue
			}

			c, err := api.toAPIChat(chat, nil, pubKey, true)
			if err != nil {
				return nil, err
			}
			result.Chats[chat.ID] = c
		}

		return result, nil
	}

	joinedCommunities, err := api.s.messenger.JoinedCommunities()
	if err != nil {
		return nil, err
	}

	found := false
	var community *communities.Community
	for _, comm := range joinedCommunities {
		if comm.IDString() == channelGroupID {
			community = comm
			found = true
			break
		}
	}

	if !found {
		spectatedCommunities, err := api.s.messenger.SpectatedCommunities()
		if err != nil {
			return nil, err
		}

		for _, comm := range spectatedCommunities {
			if comm.IDString() == channelGroupID {
				community = comm
				found = true
				break
			}
		}
	}

	if !found {
		return nil, ErrCommunityNotFound
	}

	result := &ChannelGroup{
		Type:                    Community,
		Name:                    community.Name(),
		Color:                   community.Color(),
		Images:                  make(map[string]images.IdentityImage),
		Chats:                   make(map[string]*Chat),
		Categories:              make(map[string]communities.CommunityCategory),
		Admin:                   community.IsAdmin(),
		Verified:                community.Verified(),
		Description:             community.DescriptionText(),
		IntroMessage:            community.IntroMessage(),
		OutroMessage:            community.OutroMessage(),
		Tags:                    community.Tags(),
		Permissions:             community.Description().Permissions,
		Members:                 community.Description().Members,
		CanManageUsers:          community.CanManageUsers(community.MemberIdentity()),
		Muted:                   community.Muted(),
		BanList:                 community.Description().BanList,
		Encrypted:               community.Encrypted(),
		CommunityTokensMetadata: community.Description().CommunityTokensMetadata,
	}

	for t, i := range community.Images() {
		result.Images[t] = images.IdentityImage{Name: t, Payload: i.Payload}
	}

	for _, cat := range community.Categories() {
		result.Categories[cat.CategoryId] = communities.CommunityCategory{
			ID:       cat.CategoryId,
			Name:     cat.Name,
			Position: int(cat.Position),
		}
	}

	channels := api.s.messenger.Chats()
	for _, chat := range channels {
		if chat.CommunityID == community.IDString() && chat.Active {
			c, err := api.toAPIChat(chat, community, pubKey, true)
			if err != nil {
				return nil, err
			}

			result.Chats[c.ID] = c
		}
	}

	return result, nil
}

func (api *API) GetChat(ctx context.Context, communityID types.HexBytes, chatID string) (*Chat, error) {
	pubKey := types.EncodeHex(crypto.FromECDSAPub(api.s.messenger.IdentityPublicKey()))
	messengerChat, community, err := api.getChatAndCommunity(pubKey, communityID, chatID)
	if err != nil {
		return nil, err
	}

	if messengerChat == nil {
		return nil, ErrChatNotFound
	}

	result, err := api.toAPIChat(messengerChat, community, pubKey, false)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (api *API) GetMembers(ctx context.Context, communityID types.HexBytes, chatID string) (map[string]Member, error) {
	pubKey := types.EncodeHex(crypto.FromECDSAPub(api.s.messenger.IdentityPublicKey()))
	messengerChat, community, err := api.getChatAndCommunity(pubKey, communityID, chatID)
	if err != nil {
		return nil, err
	}

	return getChatMembers(messengerChat, community, pubKey)
}

func (api *API) JoinChat(ctx context.Context, communityID types.HexBytes, chatID string) (*Chat, error) {
	if len(communityID) != 0 {
		return nil, ErrCommunitiesNotSupported
	}

	response, err := api.s.messenger.CreatePublicChat(&requests.CreatePublicChat{ID: chatID})
	if err != nil {
		return nil, err
	}

	pubKey := types.EncodeHex(crypto.FromECDSAPub(api.s.messenger.IdentityPublicKey()))

	return api.toAPIChat(response.Chats()[0], nil, pubKey, false)
}

func (api *API) toAPIChat(protocolChat *protocol.Chat, community *communities.Community, pubKey string, onlyChat bool) (*Chat, error) {
	chat := &Chat{
		ID:                       strings.TrimPrefix(protocolChat.ID, protocolChat.CommunityID),
		Name:                     protocolChat.Name,
		Description:              protocolChat.Description,
		Color:                    protocolChat.Color,
		Emoji:                    protocolChat.Emoji,
		Active:                   protocolChat.Active,
		ChatType:                 protocolChat.ChatType,
		Timestamp:                protocolChat.Timestamp,
		LastClockValue:           protocolChat.LastClockValue,
		DeletedAtClockValue:      protocolChat.DeletedAtClockValue,
		ReadMessagesAtClockValue: protocolChat.ReadMessagesAtClockValue,
		UnviewedMessagesCount:    protocolChat.UnviewedMessagesCount,
		UnviewedMentionsCount:    protocolChat.UnviewedMentionsCount,
		LastMessage:              protocolChat.LastMessage,
		MembershipUpdates:        protocolChat.MembershipUpdates,
		Alias:                    protocolChat.Alias,
		Identicon:                protocolChat.Identicon,
		Muted:                    protocolChat.Muted,
		InvitationAdmin:          protocolChat.InvitationAdmin,
		ReceivedInvitationAdmin:  protocolChat.ReceivedInvitationAdmin,
		Profile:                  protocolChat.Profile,
		CommunityID:              protocolChat.CommunityID,
		CategoryID:               protocolChat.CategoryID,
		Joined:                   protocolChat.Joined,
		SyncedTo:                 protocolChat.SyncedTo,
		SyncedFrom:               protocolChat.SyncedFrom,
		FirstMessageTimestamp:    protocolChat.FirstMessageTimestamp,
		Highlight:                protocolChat.Highlight,
		Base64Image:              protocolChat.Base64Image,
	}

	if protocolChat.OneToOne() {
		chat.Name = "" // Emptying since it contains non useful data
	}

	if !onlyChat {
		pinnedMessages, cursor, err := api.s.messenger.PinnedMessageByChatID(protocolChat.ID, "", -1)
		if err != nil {
			return nil, err
		}

		if len(pinnedMessages) != 0 {
			chat.PinnedMessages = &PinnedMessages{
				Cursor:         cursor,
				PinnedMessages: pinnedMessages,
			}
		}
	}

	err := chat.populateCommunityFields(community)
	if err != nil {
		return nil, err
	}

	if !onlyChat {
		chatMembers, err := getChatMembers(protocolChat, community, pubKey)
		if err != nil {
			return nil, err
		}
		chat.Members = chatMembers
	}

	return chat, nil
}

func getChatMembers(sourceChat *protocol.Chat, community *communities.Community, userPubKey string) (map[string]Member, error) {
	result := make(map[string]Member)
	if sourceChat != nil {
		if sourceChat.ChatType == protocol.ChatTypePrivateGroupChat && len(sourceChat.Members) > 0 {
			for _, m := range sourceChat.Members {
				result[m.ID] = Member{
					Admin:  m.Admin,
					Joined: true,
				}
			}
			return result, nil
		}

		if sourceChat.ChatType == protocol.ChatTypeOneToOne {
			result[sourceChat.ID] = Member{
				Joined: true,
			}
			result[userPubKey] = Member{
				Joined: true,
			}
			return result, nil
		}
	}

	if community != nil {
		for member, m := range community.Description().Members {
			pubKey, err := common.HexToPubkey(member)
			if err != nil {
				return nil, err
			}
			result[member] = Member{
				Roles:  m.Roles,
				Joined: community.Joined(),
				Admin:  community.IsMemberAdmin(pubKey),
			}

		}
		return result, nil
	}

	return nil, nil
}

func (api *API) getCommunityByID(id string) (*communities.Community, error) {
	communityID, err := hexutil.Decode(id)
	if err != nil {
		return nil, err
	}

	community, err := api.s.messenger.GetCommunityByID(communityID)
	if community == nil && err == nil {
		return nil, ErrCommunityNotFound
	}

	return community, err
}

func (chat *Chat) populateCommunityFields(community *communities.Community) error {
	if community == nil {
		return nil
	}

	commChat, exists := community.Chats()[chat.ID]
	if !exists {
		// Skip unknown community chats. They might be channels that were deleted
		return nil
	}

	canPost, err := community.CanMemberIdentityPost(chat.ID)
	if err != nil {
		return err
	}

	chat.CategoryID = commChat.CategoryId
	chat.Position = commChat.Position
	chat.Permissions = commChat.Permissions
	chat.Emoji = commChat.Identity.Emoji
	chat.Name = commChat.Identity.DisplayName
	chat.Description = commChat.Identity.Description
	chat.CanPost = canPost

	return nil
}

func (api *API) getChatAndCommunity(pubKey string, communityID types.HexBytes, chatID string) (*protocol.Chat, *communities.Community, error) {
	fullChatID := chatID

	if string(communityID.Bytes()) == pubKey { // Obtaining chats from personal
		communityID = []byte{}
	}

	if len(communityID) != 0 {
		id := string(communityID.Bytes())

		if chatID == "" {
			community, err := api.getCommunityByID(id)
			return nil, community, err
		}

		fullChatID = id + chatID
	}

	messengerChat := api.s.messenger.Chat(fullChatID)
	if messengerChat == nil {
		return nil, nil, ErrChatNotFound
	}

	var community *communities.Community
	if messengerChat.CommunityID != "" {
		var err error
		community, err = api.getCommunityByID(messengerChat.CommunityID)

		if err != nil {
			return nil, nil, err
		}
	}

	return messengerChat, community, nil
}

func (api *API) EditChat(ctx context.Context, communityID types.HexBytes, chatID string, name string, color string, image images.CroppedImage) (*Chat, error) {
	if len(communityID) != 0 {
		return nil, ErrCommunitiesNotSupported
	}

	chatToEdit := api.s.messenger.Chat(chatID)
	if chatToEdit == nil {
		return nil, ErrChatNotFound
	}

	if chatToEdit.ChatType != protocol.ChatTypePrivateGroupChat {
		return nil, ErrChatTypeNotSupported
	}

	response, err := api.s.messenger.EditGroupChat(ctx, chatID, name, color, image)
	if err != nil {
		return nil, err
	}

	pubKey := types.EncodeHex(crypto.FromECDSAPub(api.s.messenger.IdentityPublicKey()))
	return api.toAPIChat(response.Chats()[0], nil, pubKey, false)
}

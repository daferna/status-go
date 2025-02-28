package settings

import (
	"github.com/status-im/status-go/multiaccounts/errors"
	"github.com/status-im/status-go/protocol/protobuf"
)

var (
	AnonMetricsShouldSend = SettingField{
		reactFieldName: "anon-metrics/should-send?",
		dBColumnName:   "anon_metrics_should_send",
		valueHandler:   BoolHandler,
	}
	Appearance = SettingField{
		reactFieldName: "appearance",
		dBColumnName:   "appearance",
	}
	AutoMessageEnabled = SettingField{
		reactFieldName: "auto-message-enabled?",
		dBColumnName:   "auto_message_enabled",
		valueHandler:   BoolHandler,
	}
	BackupEnabled = SettingField{
		reactFieldName: "backup-enabled?",
		dBColumnName:   "backup_enabled",
		valueHandler:   BoolHandler,
	}
	BackupFetched = SettingField{
		reactFieldName: "backup-fetched?",
		dBColumnName:   "backup_fetched",
		valueHandler:   BoolHandler,
	}
	ChaosMode = SettingField{
		reactFieldName: "chaos-mode?",
		dBColumnName:   "chaos_mode",
		valueHandler:   BoolHandler,
	}
	Currency = SettingField{
		reactFieldName: "currency",
		dBColumnName:   "currency",
		syncProtobufFactory: &SyncProtobufFactory{
			fromInterface:     currencyProtobufFactory,
			fromStruct:        currencyProtobufFactoryStruct,
			valueFromProtobuf: StringFromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_CURRENCY,
		},
	}
	CurrentUserStatus = SettingField{
		reactFieldName: "current-user-status",
		dBColumnName:   "current_user_status",
		valueHandler:   JSONBlobHandler,
	}
	CustomBootNodes = SettingField{
		reactFieldName: "custom-bootnodes",
		dBColumnName:   "custom_bootnodes",
		valueHandler:   JSONBlobHandler,
	}
	CustomBootNodesEnabled = SettingField{
		reactFieldName: "custom-bootnodes-enabled?",
		dBColumnName:   "custom_bootnodes_enabled",
		valueHandler:   JSONBlobHandler,
	}
	DappsAddress = SettingField{
		reactFieldName: "dapps-address",
		dBColumnName:   "dapps_address",
		valueHandler:   AddressHandler,
	}
	DefaultSyncPeriod = SettingField{
		reactFieldName: "default-sync-period",
		dBColumnName:   "default_sync_period",
	}
	DisplayName = SettingField{
		reactFieldName: "display-name",
		dBColumnName:   "display_name",
		syncProtobufFactory: &SyncProtobufFactory{
			fromInterface:     displayNameProtobufFactory,
			fromStruct:        displayNameProtobufFactoryStruct,
			valueFromProtobuf: StringFromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_DISPLAY_NAME,
		},
	}
	Bio = SettingField{
		reactFieldName: "bio",
		dBColumnName:   "bio",
	}
	EIP1581Address = SettingField{
		reactFieldName: "eip1581-address",
		dBColumnName:   "eip1581_address",
		valueHandler:   AddressHandler,
	}
	Fleet = SettingField{
		reactFieldName: "fleet",
		dBColumnName:   "fleet",
	}
	GifAPIKey = SettingField{
		reactFieldName: "gifs/api-key",
		dBColumnName:   "gif_api_key",
	}
	GifFavourites = SettingField{
		reactFieldName: "gifs/favorite-gifs",
		dBColumnName:   "gif_favorites",
		valueHandler:   JSONBlobHandler,
		// TODO resolve issue 8 https://github.com/status-im/status-mobile/pull/13053#issuecomment-1065179963
		//  The reported issue is not directly related, but I suspect that gifs suffer the same issue
		syncProtobufFactory: &SyncProtobufFactory{
			inactive:          true, // Remove after issue is resolved
			fromInterface:     gifFavouritesProtobufFactory,
			fromStruct:        gifFavouritesProtobufFactoryStruct,
			valueFromProtobuf: BytesFromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_GIF_FAVOURITES,
		},
	}
	GifRecents = SettingField{
		reactFieldName: "gifs/recent-gifs",
		dBColumnName:   "gif_recents",
		valueHandler:   JSONBlobHandler,
		// TODO resolve issue 8 https://github.com/status-im/status-mobile/pull/13053#issuecomment-1065179963
		//  The reported issue is not directly related, but I suspect that gifs suffer the same issue
		syncProtobufFactory: &SyncProtobufFactory{
			inactive:          true, // Remove after issue is resolved
			fromInterface:     gifRecentsProtobufFactory,
			fromStruct:        gifRecentsProtobufFactoryStruct,
			valueFromProtobuf: BytesFromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_GIF_RECENTS,
		},
	}
	HideHomeTooltip = SettingField{
		reactFieldName: "hide-home-tooltip?",
		dBColumnName:   "hide_home_tooltip",
		valueHandler:   BoolHandler,
	}
	KeycardInstanceUID = SettingField{
		reactFieldName: "keycard-instance_uid",
		dBColumnName:   "keycard_instance_uid",
	}
	KeycardPairedOn = SettingField{
		reactFieldName: "keycard-paired_on",
		dBColumnName:   "keycard_paired_on",
	}
	KeycardPairing = SettingField{
		reactFieldName: "keycard-pairing",
		dBColumnName:   "keycard_pairing",
	}
	LastBackup = SettingField{
		reactFieldName: "last-backup",
		dBColumnName:   "last_backup",
	}
	LastUpdated = SettingField{
		reactFieldName: "last-updated",
		dBColumnName:   "last_updated",
	}
	LatestDerivedPath = SettingField{
		reactFieldName: "latest-derived-path",
		dBColumnName:   "latest_derived_path",
	}
	LinkPreviewRequestEnabled = SettingField{
		reactFieldName: "link-preview-request-enabled",
		dBColumnName:   "link_preview_request_enabled",
		valueHandler:   BoolHandler,
	}
	LinkPreviewsEnabledSites = SettingField{
		reactFieldName: "link-previews-enabled-sites",
		dBColumnName:   "link_previews_enabled_sites",
		valueHandler:   JSONBlobHandler,
	}
	LogLevel = SettingField{
		reactFieldName: "log-level",
		dBColumnName:   "log_level",
	}
	MessagesFromContactsOnly = SettingField{
		reactFieldName: "messages-from-contacts-only",
		dBColumnName:   "messages_from_contacts_only",
		valueHandler:   BoolHandler,
		syncProtobufFactory: &SyncProtobufFactory{
			fromInterface:     messagesFromContactsOnlyProtobufFactory,
			fromStruct:        messagesFromContactsOnlyProtobufFactoryStruct,
			valueFromProtobuf: BoolFromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_MESSAGES_FROM_CONTACTS_ONLY,
		},
	}
	Mnemonic = SettingField{
		reactFieldName: "mnemonic",
		dBColumnName:   "mnemonic",
	}
	MutualContactEnabled = SettingField{
		reactFieldName: "mutual-contact-enabled?",
		dBColumnName:   "mutual_contact_enabled",
		valueHandler:   BoolHandler,
	}
	Name = SettingField{
		reactFieldName: "name",
		dBColumnName:   "name",
	}
	NetworksCurrentNetwork = SettingField{
		reactFieldName: "networks/current-network",
		dBColumnName:   "current_network",
	}
	NetworksNetworks = SettingField{
		reactFieldName: "networks/networks",
		dBColumnName:   "networks",
		valueHandler:   JSONBlobHandler,
	}
	NodeConfig = SettingField{
		reactFieldName: "node-config",
		dBColumnName:   "node_config",
		valueHandler:   NodeConfigHandler,
	}
	// NotificationsEnabled - we should remove this and realated things once mobile team starts usign `settings_notifications` package
	NotificationsEnabled = SettingField{
		reactFieldName: "notifications-enabled?",
		dBColumnName:   "notifications_enabled",
		valueHandler:   BoolHandler,
	}
	OpenseaEnabled = SettingField{
		reactFieldName: "opensea-enabled?",
		dBColumnName:   "opensea_enabled",
		valueHandler:   BoolHandler,
	}
	PhotoPath = SettingField{
		reactFieldName: "photo-path",
		dBColumnName:   "photo_path",
	}
	PinnedMailservers = SettingField{
		reactFieldName: "pinned-mailservers",
		dBColumnName:   "pinned_mailservers",
		valueHandler:   JSONBlobHandler,
	}
	PreferredName = SettingField{
		reactFieldName: "preferred-name",
		dBColumnName:   "preferred_name",
		// TODO resolve issue 9 https://github.com/status-im/status-mobile/pull/13053#issuecomment-1075336559
		syncProtobufFactory: &SyncProtobufFactory{
			inactive:          true, // Remove after issue is resolved
			fromInterface:     preferredNameProtobufFactory,
			fromStruct:        preferredNameProtobufFactoryStruct,
			valueFromProtobuf: StringFromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_PREFERRED_NAME,
		},
	}
	PreviewPrivacy = SettingField{
		reactFieldName: "preview-privacy?",
		dBColumnName:   "preview_privacy",
		valueHandler:   BoolHandler,
		// TODO resolved issue 7 https://github.com/status-im/status-mobile/pull/13053#issuecomment-1065179963
		syncProtobufFactory: &SyncProtobufFactory{
			inactive:          true, // Remove after issue is resolved
			fromInterface:     previewPrivacyProtobufFactory,
			fromStruct:        previewPrivacyProtobufFactoryStruct,
			valueFromProtobuf: BoolFromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_PREVIEW_PRIVACY,
		},
	}
	ProfilePicturesShowTo = SettingField{
		reactFieldName: "profile-pictures-show-to",
		dBColumnName:   "profile_pictures_show_to",
		syncProtobufFactory: &SyncProtobufFactory{
			fromInterface:     profilePicturesShowToProtobufFactory,
			fromStruct:        profilePicturesShowToProtobufFactoryStruct,
			valueFromProtobuf: Int64FromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_PROFILE_PICTURES_SHOW_TO,
		},
	}
	ProfilePicturesVisibility = SettingField{
		reactFieldName: "profile-pictures-visibility",
		dBColumnName:   "profile_pictures_visibility",
		syncProtobufFactory: &SyncProtobufFactory{
			fromInterface:     profilePicturesVisibilityProtobufFactory,
			fromStruct:        profilePicturesVisibilityProtobufFactoryStruct,
			valueFromProtobuf: Int64FromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_PROFILE_PICTURES_VISIBILITY,
		},
	}
	PublicKey = SettingField{
		reactFieldName: "public-key",
		dBColumnName:   "public_key",
	}
	PushNotificationsBlockMentions = SettingField{
		reactFieldName: "push-notifications-block-mentions?",
		dBColumnName:   "push_notifications_block_mentions",
		valueHandler:   BoolHandler,
	}
	PushNotificationsFromContactsOnly = SettingField{
		reactFieldName: "push-notifications-from-contacts-only?",
		dBColumnName:   "push_notifications_from_contacts_only",
		valueHandler:   BoolHandler,
	}
	PushNotificationsServerEnabled = SettingField{
		reactFieldName: "push-notifications-server-enabled?",
		dBColumnName:   "push_notifications_server_enabled",
		valueHandler:   BoolHandler,
	}
	RememberSyncingChoice = SettingField{
		reactFieldName: "remember-syncing-choice?",
		dBColumnName:   "remember_syncing_choice",
		valueHandler:   BoolHandler,
	}
	RemotePushNotificationsEnabled = SettingField{
		reactFieldName: "remote-push-notifications-enabled?",
		dBColumnName:   "remote_push_notifications_enabled",
		valueHandler:   BoolHandler,
	}
	SendPushNotifications = SettingField{
		reactFieldName: "send-push-notifications?",
		dBColumnName:   "send_push_notifications",
		valueHandler:   BoolHandler,
	}
	SendStatusUpdates = SettingField{
		reactFieldName: "send-status-updates?",
		dBColumnName:   "send_status_updates",
		valueHandler:   BoolHandler,
		// TODO resolve issue 10 https://github.com/status-im/status-mobile/pull/13053#issuecomment-1075352256
		syncProtobufFactory: &SyncProtobufFactory{
			inactive:          true, // Remove after issue is resolved
			fromInterface:     sendStatusUpdatesProtobufFactory,
			fromStruct:        sendStatusUpdatesProtobufFactoryStruct,
			valueFromProtobuf: BoolFromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_SEND_STATUS_UPDATES,
		},
	}
	StickersPacksInstalled = SettingField{
		reactFieldName: "stickers/packs-installed",
		dBColumnName:   "stickers_packs_installed",
		valueHandler:   JSONBlobHandler,
		syncProtobufFactory: &SyncProtobufFactory{
			inactive:          true, // TODO current version of stickers introduces a regression on deleting sticker packs
			fromInterface:     stickersPacksInstalledProtobufFactory,
			fromStruct:        stickersPacksInstalledProtobufFactoryStruct,
			valueFromProtobuf: BytesFromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_STICKERS_PACKS_INSTALLED,
		},
	}
	StickersPacksPending = SettingField{
		reactFieldName: "stickers/packs-pending",
		dBColumnName:   "stickers_packs_pending",
		valueHandler:   JSONBlobHandler,
		syncProtobufFactory: &SyncProtobufFactory{
			inactive:          true, // TODO current version of stickers introduces a regression on deleting sticker packs
			fromInterface:     stickersPacksPendingProtobufFactory,
			fromStruct:        stickersPacksPendingProtobufFactoryStruct,
			valueFromProtobuf: BytesFromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_STICKERS_PACKS_PENDING,
		},
	}
	StickersRecentStickers = SettingField{
		reactFieldName: "stickers/recent-stickers",
		dBColumnName:   "stickers_recent_stickers",
		valueHandler:   JSONBlobHandler,
		syncProtobufFactory: &SyncProtobufFactory{
			inactive:          true, // TODO current version of stickers introduces a regression on deleting sticker packs
			fromInterface:     stickersRecentStickersProtobufFactory,
			fromStruct:        stickersRecentStickersProtobufFactoryStruct,
			valueFromProtobuf: BytesFromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_STICKERS_RECENT_STICKERS,
		},
	}
	SyncingOnMobileNetwork = SettingField{
		reactFieldName: "syncing-on-mobile-network?",
		dBColumnName:   "syncing_on_mobile_network",
		valueHandler:   BoolHandler,
	}
	TelemetryServerURL = SettingField{
		reactFieldName: "telemetry-server-url",
		dBColumnName:   "telemetry_server_url",
	}
	TestNetworksEnabled = SettingField{
		reactFieldName: "test-networks-enabled?",
		dBColumnName:   "test_networks_enabled",
		valueHandler:   BoolHandler,
	}
	UseMailservers = SettingField{
		reactFieldName: "use-mailservers?",
		dBColumnName:   "use_mailservers",
		valueHandler:   BoolHandler,
	}
	Usernames = SettingField{
		reactFieldName: "usernames",
		dBColumnName:   "usernames",
		valueHandler:   JSONBlobHandler,
	}
	WakuBloomFilterMode = SettingField{
		reactFieldName: "waku-bloom-filter-mode",
		dBColumnName:   "waku_bloom_filter_mode",
		valueHandler:   BoolHandler,
	}
	WalletSetUpPassed = SettingField{
		reactFieldName: "wallet-set-up-passed?",
		dBColumnName:   "wallet_set_up_passed",
		valueHandler:   BoolHandler,
	}
	WalletVisibleTokens = SettingField{
		reactFieldName: "wallet/visible-tokens",
		dBColumnName:   "wallet_visible_tokens",
		valueHandler:   JSONBlobHandler,
	}
	WebviewAllowPermissionRequests = SettingField{
		reactFieldName: "webview-allow-permission-requests?",
		dBColumnName:   "webview_allow_permission_requests",
		valueHandler:   BoolHandler,
	}
	WalletRootAddress = SettingField{
		reactFieldName: "wallet-root-address",
		dBColumnName:   "wallet_root_address",
		valueHandler:   AddressHandler,
	}
	MasterAddress = SettingField{
		reactFieldName: "address",
		dBColumnName:   "address",
		valueHandler:   AddressHandler,
	}

	SettingFieldRegister = []SettingField{
		AnonMetricsShouldSend,
		Appearance,
		AutoMessageEnabled,
		BackupEnabled,
		BackupFetched,
		ChaosMode,
		Currency,
		CurrentUserStatus,
		CustomBootNodes,
		CustomBootNodesEnabled,
		DappsAddress,
		DefaultSyncPeriod,
		DisplayName,
		EIP1581Address,
		Fleet,
		GifAPIKey,
		GifFavourites,
		GifRecents,
		HideHomeTooltip,
		KeycardInstanceUID,
		KeycardPairedOn,
		KeycardPairing,
		LastBackup,
		LastUpdated,
		LatestDerivedPath,
		LinkPreviewRequestEnabled,
		LinkPreviewsEnabledSites,
		LogLevel,
		MessagesFromContactsOnly,
		Mnemonic,
		MutualContactEnabled,
		Name,
		NetworksCurrentNetwork,
		NetworksNetworks,
		NodeConfig,
		NotificationsEnabled,
		OpenseaEnabled,
		PhotoPath,
		PinnedMailservers,
		PreferredName,
		PreviewPrivacy,
		ProfilePicturesShowTo,
		ProfilePicturesVisibility,
		PublicKey,
		PushNotificationsBlockMentions,
		PushNotificationsFromContactsOnly,
		PushNotificationsServerEnabled,
		RememberSyncingChoice,
		RemotePushNotificationsEnabled,
		SendPushNotifications,
		SendStatusUpdates,
		StickersPacksInstalled,
		StickersPacksPending,
		StickersRecentStickers,
		SyncingOnMobileNetwork,
		TelemetryServerURL,
		TestNetworksEnabled,
		UseMailservers,
		Usernames,
		WakuBloomFilterMode,
		WalletRootAddress,
		WalletSetUpPassed,
		WalletVisibleTokens,
		WebviewAllowPermissionRequests,
	}
)

func GetFieldFromProtobufType(pbt protobuf.SyncSetting_Type) (SettingField, error) {
	if pbt == protobuf.SyncSetting_UNKNOWN {
		return SettingField{}, errors.ErrUnrecognisedSyncSettingProtobufType
	}

	for _, s := range SettingFieldRegister {
		if s.SyncProtobufFactory() == nil {
			continue
		}
		if s.SyncProtobufFactory().SyncSettingProtobufType() == pbt {
			return s, nil
		}
	}

	return SettingField{}, errors.ErrUnrecognisedSyncSettingProtobufType
}

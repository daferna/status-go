package wakusync

import (
	"encoding/json"

	"github.com/status-im/status-go/multiaccounts/keypairs"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/protobuf"
)

type WakuBackedUpDataResponse struct {
	FetchingDataProgress map[string]protobuf.FetchingBackedUpDataDetails // key represents the data/section backup details refer to
	Profile              *BackedUpProfile
	Setting              *settings.SyncSettingField
	Keycards             []*keypairs.KeyPair
}

func (sfwr *WakuBackedUpDataResponse) MarshalJSON() ([]byte, error) {
	responseItem := struct {
		FetchingDataProgress map[string]FetchingBackupedDataDetails `json:"fetchingBackedUpDataProgress,omitempty"`
		Profile              *BackedUpProfile                       `json:"backedUpProfile,omitempty"`
		Setting              *settings.SyncSettingField             `json:"backedUpSettings,omitempty"`
		Keycards             []*keypairs.KeyPair                    `json:"backedUpKeycards,omitempty"`
	}{
		Profile:  sfwr.Profile,
		Setting:  sfwr.Setting,
		Keycards: sfwr.Keycards,
	}

	responseItem.FetchingDataProgress = sfwr.FetchingBackedUpDataDetails()

	return json.Marshal(responseItem)
}

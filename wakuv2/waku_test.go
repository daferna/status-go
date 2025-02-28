package wakuv2

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"

	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/wakuv2/common"
)

var testENRBootstrap = "enrtree://AOGECG2SPND25EEFMAJ5WF3KSGJNSGV356DSTL2YVLLZWIV6SAYBM@prod.nodes.status.im"

func TestDiscoveryV5(t *testing.T) {
	config := &Config{}
	config.EnableDiscV5 = true
	config.DiscV5BootstrapNodes = []string{testENRBootstrap}
	config.DiscoveryLimit = 20
	config.UDPPort = 9001
	w, err := New("", "", config, nil, nil, nil)
	require.NoError(t, err)

	require.NoError(t, w.Start())

	err = tt.RetryWithBackOff(func() error {
		if len(w.Peers()) == 0 {
			return errors.New("no peers discovered")
		}
		return nil
	})

	require.NoError(t, err)

	require.NotEqual(t, 0, len(w.Peers()))
	require.NoError(t, w.Stop())
}

func TestRestartDiscoveryV5(t *testing.T) {
	config := &Config{}
	config.EnableDiscV5 = true
	// Use wrong discv5 bootstrap address, to simulate being offline
	config.DiscV5BootstrapNodes = []string{"enrtree://AOGECG2SPND25EEFMAJ5WF3KSGJNSGV356DSTL2YVLLZWIV6SAYBM@1.1.1.2"}
	config.DiscoveryLimit = 20
	config.UDPPort = 9002
	w, err := New("", "", config, nil, nil, nil)
	require.NoError(t, err)

	require.NoError(t, w.Start())

	require.False(t, w.seededBootnodesForDiscV5)

	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 2 * time.Second
	}

	// Sanity check, not great, but it's probably helpful
	err = tt.RetryWithBackOff(func() error {
		if len(w.Peers()) == 0 {
			return errors.New("no peers discovered")
		}
		return nil
	}, options)

	require.Error(t, err)

	w.discV5BootstrapNodes = []string{testENRBootstrap}

	options = func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 30 * time.Second
	}

	err = tt.RetryWithBackOff(func() error {
		if len(w.Peers()) == 0 {
			return errors.New("no peers discovered")
		}
		return nil
	}, options)
	require.NoError(t, err)

	require.True(t, w.seededBootnodesForDiscV5)
	require.NotEqual(t, 0, len(w.Peers()))
	require.NoError(t, w.Stop())
}

func TestBasicWakuV2(t *testing.T) {
	enrTreeAddress := testENRBootstrap
	envEnrTreeAddress := os.Getenv("ENRTREE_ADDRESS")
	if envEnrTreeAddress != "" {
		enrTreeAddress = envEnrTreeAddress
	}

	config := &Config{}
	config.Port = 0
	config.EnableDiscV5 = true
	config.DiscV5BootstrapNodes = []string{enrTreeAddress}
	config.DiscoveryLimit = 20
	config.UDPPort = 9001
	config.WakuNodes = []string{enrTreeAddress}
	w, err := New("", "", config, nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, w.Start())

	// DNSDiscovery
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	discoveredNodes, err := dnsdisc.RetrieveNodes(ctx, enrTreeAddress)
	require.NoError(t, err)

	// Peer used for retrieving history
	r, err := rand.Int(rand.Reader, big.NewInt(int64(len(discoveredNodes))))
	require.NoError(t, err)

	storeNode := discoveredNodes[int(r.Int64())]

	// Wait for some peers to be discovered
	time.Sleep(3 * time.Second)

	// At least 3 peers should have been discovered
	require.Greater(t, w.PeerCount(), 3)

	filter := &common.Filter{
		Messages: common.NewMemoryMessageStore(),
		Topics: [][]byte{
			{1, 2, 3, 4},
		},
	}

	_, err = w.Subscribe(filter)
	require.NoError(t, err)

	msgTimestamp := w.timestamp()
	contentTopic := common.BytesToTopic(filter.Topics[0])

	_, err = w.Send(&pb.WakuMessage{
		Payload:      []byte{1, 2, 3, 4, 5},
		ContentTopic: contentTopic.ContentTopic(),
		Version:      0,
		Timestamp:    msgTimestamp,
	})
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	messages := filter.Retrieve()
	require.Len(t, messages, 1)

	timestampInSeconds := msgTimestamp / int64(time.Second)
	storeResult, err := w.query(context.Background(), storeNode.PeerID, []common.TopicType{contentTopic}, uint64(timestampInSeconds-20), uint64(timestampInSeconds+20), []store.HistoryRequestOption{})
	require.NoError(t, err)
	require.NotZero(t, len(storeResult.Messages))
}

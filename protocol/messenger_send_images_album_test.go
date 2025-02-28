package protocol

import (
	"context"
	"crypto/ecdsa"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"

	// "github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestMessengerSendImagesAlbumSuite(t *testing.T) {
	suite.Run(t, new(MessengerSendImagesAlbumSuite))
}

type MessengerSendImagesAlbumSuite struct {
	suite.Suite
	m          *Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerSendImagesAlbumSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.m = s.newMessenger()
	s.privateKey = s.m.identity
	_, err := s.m.Start()
	s.Require().NoError(err)

}

func (s *MessengerSendImagesAlbumSuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
}

func (s *MessengerSendImagesAlbumSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

func buildImageWithoutAlbumIDMessage(s *MessengerSendImagesAlbumSuite, chat Chat) *common.Message {
	file, err := os.Open("../_assets/tests/test.jpg")
	s.Require().NoError(err)
	defer file.Close()

	payload, err := ioutil.ReadAll(file)
	s.Require().NoError(err)

	clock, timestamp := chat.NextClockAndTimestamp(&testTimeSource{})
	message := &common.Message{}
	message.ChatId = chat.ID
	message.Clock = clock
	message.Timestamp = timestamp
	message.WhisperTimestamp = clock
	message.LocalChatID = chat.ID
	message.MessageType = protobuf.MessageType_ONE_TO_ONE
	message.ContentType = protobuf.ChatMessage_IMAGE

	image := protobuf.ImageMessage{
		Payload: payload,
		Type:    protobuf.ImageType_JPEG,
		Width:   1200,
		Height:  1000,
	}
	message.Payload = &protobuf.ChatMessage_Image{Image: &image}
	return message
}

func (s *MessengerSendImagesAlbumSuite) TestAlbumImageMessagesSend() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("Our 1TO1", &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(ourChat)
	s.Require().NoError(err)

	const messageCount = 3
	var album []*common.Message

	for i := 0; i < messageCount; i++ {
		album = append(album, buildImageWithoutAlbumIDMessage(s, *ourChat))
	}

	err = s.m.SaveChat(ourChat)
	s.NoError(err)
	response, err := s.m.SendChatMessages(context.Background(), album)
	s.NoError(err)
	s.Require().Equal(messageCount, len(response.Messages()), "it returns the messages")
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), messageCount)

	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.messages) > 0 },
		"no messages",
	)

	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), messageCount)

	for _, message := range response.Messages() {
		image := message.GetImage()
		s.Require().NotNil(image)
		s.Require().NotEmpty(image.AlbumId)
	}
}

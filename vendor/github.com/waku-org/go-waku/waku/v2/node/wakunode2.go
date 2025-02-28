package node

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/backoff"
	ws "github.com/libp2p/go-libp2p/p2p/transport/websocket"
	ma "github.com/multiformats/go-multiaddr"
	"go.opencensus.io/stats"

	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/try"
	v2 "github.com/waku-org/go-waku/waku/v2"
	"github.com/waku-org/go-waku/waku/v2/discv5"
	"github.com/waku-org/go-waku/waku/v2/metrics"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/filterv2"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/peer_exchange"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	"github.com/waku-org/go-waku/waku/v2/protocol/swap"
	"github.com/waku-org/go-waku/waku/v2/rendezvous"
	"github.com/waku-org/go-waku/waku/v2/timesource"

	"github.com/waku-org/go-waku/waku/v2/utils"
)

type Peer struct {
	ID        peer.ID        `json:"peerID"`
	Protocols []protocol.ID  `json:"protocols"`
	Addrs     []ma.Multiaddr `json:"addrs"`
	Connected bool           `json:"connected"`
}

type storeFactory func(w *WakuNode) store.Store

type MembershipKeyPair = struct {
	IDKey        [32]byte `json:"idKey"`
	IDCommitment [32]byte `json:"idCommitment"`
}

type RLNRelay interface {
	MembershipKeyPair() *MembershipKeyPair
	MembershipIndex() uint
	MembershipContractAddress() common.Address
	AppendRLNProof(msg *pb.WakuMessage, senderEpochTime time.Time) error
	Stop()
}

type WakuNode struct {
	host       host.Host
	opts       *WakuNodeParameters
	log        *zap.Logger
	timesource timesource.Timesource

	relay         Service
	lightPush     Service
	swap          Service
	peerConnector PeerConnectorService
	discoveryV5   Service
	peerExchange  Service
	rendezvous    Service
	filter        ReceptorService
	filterV2Full  ReceptorService
	filterV2Light Service
	store         ReceptorService
	rlnRelay      RLNRelay

	wakuFlag utils.WakuEnrBitfield

	localNode *enode.LocalNode

	bcaster v2.Broadcaster

	connectionNotif        ConnectionNotifier
	protocolEventSub       event.Subscription
	identificationEventSub event.Subscription
	addressChangesSub      event.Subscription
	enrChangeCh            chan struct{}

	keepAliveMutex sync.Mutex
	keepAliveFails map[peer.ID]int

	cancel context.CancelFunc
	wg     *sync.WaitGroup

	// Channel passed to WakuNode constructor
	// receiving connection status notifications
	connStatusChan chan ConnStatus

	storeFactory storeFactory
}

func defaultStoreFactory(w *WakuNode) store.Store {
	return store.NewWakuStore(w.host, w.swap, w.opts.messageProvider, w.timesource, w.log)
}

// New is used to instantiate a WakuNode using a set of WakuNodeOptions
func New(opts ...WakuNodeOption) (*WakuNode, error) {
	params := new(WakuNodeParameters)
	params.libP2POpts = DefaultLibP2POptions

	opts = append(DefaultWakuNodeOptions, opts...)
	for _, opt := range opts {
		err := opt(params)
		if err != nil {
			return nil, err
		}
	}

	if params.logger == nil {
		params.logger = utils.Logger()
		golog.SetAllLoggers(params.logLevel)
	}

	if params.privKey == nil {
		prvKey, err := crypto.GenerateKey()
		if err != nil {
			return nil, err
		}
		params.privKey = prvKey
	}

	if params.enableWSS {
		params.libP2POpts = append(params.libP2POpts, libp2p.Transport(ws.New, ws.WithTLSConfig(params.tlsConfig)))
	} else {
		// Enable WS transport by default
		params.libP2POpts = append(params.libP2POpts, libp2p.Transport(ws.New))
	}

	// Setting default host address if none was provided
	if params.hostAddr == nil {
		err := WithHostAddress(&net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: 0})(params)
		if err != nil {
			return nil, err
		}
	}
	if len(params.multiAddr) > 0 {
		params.libP2POpts = append(params.libP2POpts, libp2p.ListenAddrs(params.multiAddr...))
	}

	params.libP2POpts = append(params.libP2POpts, params.Identity())

	if params.addressFactory != nil {
		params.libP2POpts = append(params.libP2POpts, libp2p.AddrsFactory(params.addressFactory))
	}

	host, err := libp2p.New(params.libP2POpts...)
	if err != nil {
		return nil, err
	}

	w := new(WakuNode)
	w.bcaster = v2.NewBroadcaster(1024)
	w.host = host
	w.opts = params
	w.log = params.logger.Named("node2")
	w.wg = &sync.WaitGroup{}
	w.keepAliveFails = make(map[peer.ID]int)
	w.wakuFlag = utils.NewWakuEnrBitfield(w.opts.enableLightPush, w.opts.enableFilter, w.opts.enableStore, w.opts.enableRelay)

	if params.enableNTP {
		w.timesource = timesource.NewNTPTimesource(w.opts.ntpURLs, w.log)
	} else {
		w.timesource = timesource.NewDefaultClock()
	}

	w.localNode, err = w.newLocalnode(w.opts.privKey)
	if err != nil {
		w.log.Error("creating localnode", zap.Error(err))
	}

	// Setup peer connection strategy
	cacheSize := 600
	rngSrc := rand.NewSource(rand.Int63())
	minBackoff, maxBackoff := time.Second*30, time.Hour
	bkf := backoff.NewExponentialBackoff(minBackoff, maxBackoff, backoff.FullJitter, time.Second, 5.0, 0, rand.New(rngSrc))
	w.peerConnector, err = v2.NewPeerConnectionStrategy(host, cacheSize, w.opts.discoveryMinPeers, network.DialPeerTimeout, bkf, w.log)
	if err != nil {
		w.log.Error("creating peer connection strategy", zap.Error(err))
	}

	if w.opts.enableDiscV5 {
		err := w.mountDiscV5()
		if err != nil {
			return nil, err
		}
	}

	w.peerExchange, err = peer_exchange.NewWakuPeerExchange(w.host, w.DiscV5(), w.peerConnector, w.log)
	if err != nil {
		return nil, err
	}

	w.rendezvous = rendezvous.NewRendezvous(w.host, w.opts.rendezvousDB, w.peerConnector, w.log)
	w.relay = relay.NewWakuRelay(w.host, w.bcaster, w.opts.minRelayPeersToPublish, w.timesource, w.log, w.opts.wOpts...)
	w.filter = filter.NewWakuFilter(w.host, w.bcaster, w.opts.isFilterFullNode, w.timesource, w.log, w.opts.filterOpts...)
	w.filterV2Full = filterv2.NewWakuFilterFullnode(w.host, w.bcaster, w.timesource, w.log, w.opts.filterV2Opts...)
	w.filterV2Light = filterv2.NewWakuFilterLightnode(w.host, w.bcaster, w.timesource, w.log)
	w.lightPush = lightpush.NewWakuLightPush(w.host, w.Relay(), w.log)

	if w.opts.enableSwap {
		w.swap = swap.NewWakuSwap(w.log, []swap.SwapOption{
			swap.WithMode(w.opts.swapMode),
			swap.WithThreshold(w.opts.swapPaymentThreshold, w.opts.swapDisconnectThreshold),
		}...)
	}

	if params.storeFactory != nil {
		w.storeFactory = params.storeFactory
	} else {
		w.storeFactory = defaultStoreFactory
	}

	if w.protocolEventSub, err = host.EventBus().Subscribe(new(event.EvtPeerProtocolsUpdated)); err != nil {
		return nil, err
	}

	if w.identificationEventSub, err = host.EventBus().Subscribe(new(event.EvtPeerIdentificationCompleted)); err != nil {
		return nil, err
	}

	if w.addressChangesSub, err = host.EventBus().Subscribe(new(event.EvtLocalAddressesUpdated)); err != nil {
		return nil, err
	}

	if params.connStatusC != nil {
		w.connStatusChan = params.connStatusC
	}

	return w, nil
}

func (w *WakuNode) watchMultiaddressChanges(ctx context.Context) {
	defer w.wg.Done()

	addrs := w.ListenAddresses()
	first := make(chan struct{}, 1)
	first <- struct{}{}
	for {
		select {
		case <-ctx.Done():
			return
		case <-first:
			w.log.Info("listening", logging.MultiAddrs("multiaddr", addrs...))
			w.enrChangeCh <- struct{}{}
		case <-w.addressChangesSub.Out():
			newAddrs := w.ListenAddresses()
			diff := false
			if len(addrs) != len(newAddrs) {
				diff = true
			} else {
				for i := range newAddrs {
					if addrs[i].String() != newAddrs[i].String() {
						diff = true
						break
					}
				}
			}
			if diff {
				addrs = newAddrs
				w.log.Info("listening addresses update received", logging.MultiAddrs("multiaddr", addrs...))
				_ = w.setupENR(ctx, addrs)
				w.enrChangeCh <- struct{}{}
			}
		}
	}
}

// Start initializes all the protocols that were setup in the WakuNode
func (w *WakuNode) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel

	w.connectionNotif = NewConnectionNotifier(ctx, w.host, w.log)
	w.host.Network().Notify(w.connectionNotif)

	w.enrChangeCh = make(chan struct{}, 10)

	w.wg.Add(3)
	go w.connectednessListener(ctx)
	go w.watchMultiaddressChanges(ctx)
	go w.watchENRChanges(ctx)

	if w.opts.keepAliveInterval > time.Duration(0) {
		w.wg.Add(1)
		go w.startKeepAlive(ctx, w.opts.keepAliveInterval)
	}

	err := w.peerConnector.Start(ctx)
	if err != nil {
		return err
	}

	if w.opts.enableNTP {
		err := w.timesource.Start(ctx)
		if err != nil {
			return err
		}
	}

	if w.opts.enableRelay {
		err := w.relay.Start(ctx)
		if err != nil {
			return err
		}

		if !w.opts.noDefaultWakuTopic {
			sub, err := w.Relay().Subscribe(ctx)
			if err != nil {
				return err
			}

			w.Broadcaster().Unregister(&relay.DefaultWakuTopic, sub.C)
		}
	}

	w.store = w.storeFactory(w)
	if w.opts.enableStore {
		err := w.startStore(ctx)
		if err != nil {
			return err
		}

		w.log.Info("Subscribing store to broadcaster")
		w.bcaster.Register(nil, w.store.MessageChannel())
	}

	if w.opts.enableLightPush {
		if err := w.lightPush.Start(ctx); err != nil {
			return err
		}
	}

	if w.opts.enableFilter {
		err := w.filter.Start(ctx)
		if err != nil {
			return err
		}

		w.log.Info("Subscribing filter to broadcaster")
		w.bcaster.Register(nil, w.filter.MessageChannel())
	}

	if w.opts.enableFilterV2FullNode {
		err := w.filterV2Full.Start(ctx)
		if err != nil {
			return err
		}

		w.log.Info("Subscribing filterV2 to broadcaster")
		w.bcaster.Register(nil, w.filterV2Full.MessageChannel())
	}

	if w.opts.enableFilterV2LightNode {
		err := w.filterV2Light.Start(ctx)
		if err != nil {
			return err
		}
	}

	err = w.setupENR(ctx, w.ListenAddresses())
	if err != nil {
		return err
	}

	if w.opts.enablePeerExchange {
		err := w.peerExchange.Start(ctx)
		if err != nil {
			return err
		}
	}

	if w.opts.enableRendezvous {
		err := w.rendezvous.Start(ctx)
		if err != nil {
			return err
		}
	}

	if w.opts.enableRLN {
		err = w.mountRlnRelay(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// Stop stops the WakuNode and closess all connections to the host
func (w *WakuNode) Stop() {
	if w.cancel == nil {
		return
	}

	w.cancel()

	w.bcaster.Close()

	defer w.connectionNotif.Close()
	defer w.protocolEventSub.Close()
	defer w.identificationEventSub.Close()
	defer w.addressChangesSub.Close()

	if w.opts.enableRendezvous {
		w.rendezvous.Stop()
	}

	w.relay.Stop()
	w.lightPush.Stop()
	w.store.Stop()
	w.filter.Stop()
	w.filterV2Full.Stop()
	w.peerExchange.Stop()

	if w.opts.enableDiscV5 {
		w.discoveryV5.Stop()
	}

	w.peerConnector.Stop()

	_ = w.stopRlnRelay()

	w.timesource.Stop()

	w.host.Close()

	w.wg.Wait()

	close(w.enrChangeCh)
}

// Host returns the libp2p Host used by the WakuNode
func (w *WakuNode) Host() host.Host {
	return w.host
}

// ID returns the base58 encoded ID from the host
func (w *WakuNode) ID() string {
	return w.host.ID().Pretty()
}

func (w *WakuNode) watchENRChanges(ctx context.Context) {
	defer w.wg.Done()

	var prevNodeVal string
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.enrChangeCh:
			if w.localNode != nil {
				currNodeVal := w.localNode.Node().String()
				if prevNodeVal != currNodeVal {
					if prevNodeVal == "" {
						w.log.Info("enr record", logging.ENode("enr", w.localNode.Node()))
					} else {
						w.log.Info("new enr record", logging.ENode("enr", w.localNode.Node()))
					}
					prevNodeVal = currNodeVal
				}
			}
		}
	}
}

// ListenAddresses returns all the multiaddresses used by the host
func (w *WakuNode) ListenAddresses() []ma.Multiaddr {
	hostInfo, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", w.host.ID().Pretty()))
	var result []ma.Multiaddr
	for _, addr := range w.host.Addrs() {
		result = append(result, addr.Encapsulate(hostInfo))
	}
	return result
}

// ENR returns the ENR address of the node
func (w *WakuNode) ENR() *enode.Node {
	return w.localNode.Node()
}

// Timesource returns the timesource used by this node to obtain the current wall time
// Depending on the configuration it will be the local time or a ntp syncd time
func (w *WakuNode) Timesource() timesource.Timesource {
	return w.timesource
}

// Relay is used to access any operation related to Waku Relay protocol
func (w *WakuNode) Relay() *relay.WakuRelay {
	if result, ok := w.relay.(*relay.WakuRelay); ok {
		return result
	}
	return nil
}

// Store is used to access any operation related to Waku Store protocol
func (w *WakuNode) Store() store.Store {
	return w.store.(store.Store)
}

// Filter is used to access any operation related to Waku Filter protocol
func (w *WakuNode) Filter() *filter.WakuFilter {
	if result, ok := w.filter.(*filter.WakuFilter); ok {
		return result
	}
	return nil
}

// FilterV2 is used to access any operation related to Waku Filter protocol
func (w *WakuNode) FilterV2() *filterv2.WakuFilterLightnode {
	if result, ok := w.filterV2Light.(*filterv2.WakuFilterLightnode); ok {
		return result
	}
	return nil
}

// Lightpush is used to access any operation related to Waku Lightpush protocol
func (w *WakuNode) Lightpush() *lightpush.WakuLightPush {
	if result, ok := w.lightPush.(*lightpush.WakuLightPush); ok {
		return result
	}
	return nil
}

// DiscV5 is used to access any operation related to DiscoveryV5
func (w *WakuNode) DiscV5() *discv5.DiscoveryV5 {
	if result, ok := w.discoveryV5.(*discv5.DiscoveryV5); ok {
		return result
	}
	return nil
}

// PeerExchange is used to access any operation related to Peer Exchange
func (w *WakuNode) PeerExchange() *peer_exchange.WakuPeerExchange {
	if result, ok := w.peerExchange.(*peer_exchange.WakuPeerExchange); ok {
		return result
	}
	return nil
}

// Broadcaster is used to access the message broadcaster that is used to push
// messages to different protocols
func (w *WakuNode) Broadcaster() v2.Broadcaster {
	return w.bcaster
}

// Publish will attempt to publish a message via WakuRelay if there are enough
// peers available, otherwise it will attempt to publish via Lightpush protocol
func (w *WakuNode) Publish(ctx context.Context, msg *pb.WakuMessage) error {
	if !w.opts.enableLightPush && !w.opts.enableRelay {
		return errors.New("cannot publish message, relay and lightpush are disabled")
	}

	hash := msg.Hash(relay.DefaultWakuTopic)
	err := try.Do(func(attempt int) (bool, error) {
		var err error

		relay := w.Relay()
		lightpush := w.Lightpush()

		if relay == nil || !relay.EnoughPeersToPublish() {
			w.log.Debug("publishing message via lightpush", logging.HexBytes("hash", hash))
			_, err = lightpush.Publish(ctx, msg)
		} else {
			w.log.Debug("publishing message via relay", logging.HexBytes("hash", hash))
			_, err = relay.Publish(ctx, msg)
		}

		return attempt < maxPublishAttempt, err
	})

	return err
}

func (w *WakuNode) mountDiscV5() error {
	discV5Options := []discv5.DiscoveryV5Option{
		discv5.WithBootnodes(w.opts.discV5bootnodes),
		discv5.WithUDPPort(w.opts.udpPort),
		discv5.WithAutoUpdate(w.opts.discV5autoUpdate),
	}

	if w.opts.advertiseAddrs != nil {
		discV5Options = append(discV5Options, discv5.WithAdvertiseAddr(w.opts.advertiseAddrs))
	}

	var err error
	w.discoveryV5, err = discv5.NewDiscoveryV5(w.Host(), w.opts.privKey, w.localNode, w.peerConnector, w.log, discV5Options...)

	return err
}

func (w *WakuNode) startStore(ctx context.Context) error {
	err := w.store.Start(ctx)
	if err != nil {
		w.log.Error("starting store", zap.Error(err))
		return err
	}

	if len(w.opts.resumeNodes) != 0 {
		// TODO: extract this to a function and run it when you go offline
		// TODO: determine if a store is listening to a topic

		var peerIDs []peer.ID
		for _, n := range w.opts.resumeNodes {
			pID, err := w.AddPeer(n, store.StoreID_v20beta4)
			if err != nil {
				w.log.Warn("adding peer to peerstore", logging.MultiAddrs("peer", n), zap.Error(err))
			}
			peerIDs = append(peerIDs, pID)
		}

		if !w.opts.noDefaultWakuTopic {
			w.wg.Add(1)
			go func() {
				defer w.wg.Done()

				ctxWithTimeout, ctxCancel := context.WithTimeout(ctx, 20*time.Second)
				defer ctxCancel()
				if _, err := w.store.(store.Store).Resume(ctxWithTimeout, string(relay.DefaultWakuTopic), peerIDs); err != nil {
					w.log.Error("Could not resume history", zap.Error(err))
					time.Sleep(10 * time.Second)
				}
			}()
		}
	}
	return nil
}

func (w *WakuNode) addPeer(info *peer.AddrInfo, protocols ...protocol.ID) error {
	w.log.Info("adding peer to peerstore", logging.HostID("peer", info.ID))
	w.host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	err := w.host.Peerstore().AddProtocols(info.ID, protocols...)
	if err != nil {
		return err
	}

	return nil
}

// AddPeer is used to add a peer and the protocols it support to the node peerstore
func (w *WakuNode) AddPeer(address ma.Multiaddr, protocols ...protocol.ID) (peer.ID, error) {
	info, err := peer.AddrInfoFromP2pAddr(address)
	if err != nil {
		return "", err
	}

	return info.ID, w.addPeer(info, protocols...)
}

// DialPeerWithMultiAddress is used to connect to a peer using a multiaddress
func (w *WakuNode) DialPeerWithMultiAddress(ctx context.Context, address ma.Multiaddr) error {
	info, err := peer.AddrInfoFromP2pAddr(address)
	if err != nil {
		return err
	}

	return w.connect(ctx, *info)
}

// DialPeer is used to connect to a peer using a string containing a multiaddress
func (w *WakuNode) DialPeer(ctx context.Context, address string) error {
	p, err := ma.NewMultiaddr(address)
	if err != nil {
		return err
	}

	info, err := peer.AddrInfoFromP2pAddr(p)
	if err != nil {
		return err
	}

	return w.connect(ctx, *info)
}

func (w *WakuNode) connect(ctx context.Context, info peer.AddrInfo) error {
	err := w.host.Connect(ctx, info)
	if err != nil {
		return err
	}

	stats.Record(ctx, metrics.Dials.M(1))
	return nil
}

// DialPeerByID is used to connect to an already known peer
func (w *WakuNode) DialPeerByID(ctx context.Context, peerID peer.ID) error {
	info := w.host.Peerstore().PeerInfo(peerID)
	return w.connect(ctx, info)
}

// ClosePeerByAddress is used to disconnect from a peer using its multiaddress
func (w *WakuNode) ClosePeerByAddress(address string) error {
	p, err := ma.NewMultiaddr(address)
	if err != nil {
		return err
	}

	// Extract the peer ID from the multiaddr.
	info, err := peer.AddrInfoFromP2pAddr(p)
	if err != nil {
		return err
	}

	return w.ClosePeerById(info.ID)
}

// ClosePeerById is used to close a connection to a peer
func (w *WakuNode) ClosePeerById(id peer.ID) error {
	err := w.host.Network().ClosePeer(id)
	if err != nil {
		return err
	}
	return nil
}

// PeerCount return the number of connected peers
func (w *WakuNode) PeerCount() int {
	return len(w.host.Network().Peers())
}

// PeerStats returns a list of peers and the protocols supported by them
func (w *WakuNode) PeerStats() PeerStats {
	p := make(PeerStats)
	for _, peerID := range w.host.Network().Peers() {
		protocols, err := w.host.Peerstore().GetProtocols(peerID)
		if err != nil {
			continue
		}
		p[peerID] = protocols
	}
	return p
}

// Set the bootnodes on discv5
func (w *WakuNode) SetDiscV5Bootnodes(nodes []*enode.Node) error {
	w.opts.discV5bootnodes = nodes
	return w.DiscV5().SetBootnodes(nodes)
}

// Peers return the list of peers, addresses, protocols supported and connection status
func (w *WakuNode) Peers() ([]*Peer, error) {
	var peers []*Peer
	for _, peerId := range w.host.Peerstore().Peers() {
		connected := w.host.Network().Connectedness(peerId) == network.Connected
		protocols, err := w.host.Peerstore().GetProtocols(peerId)
		if err != nil {
			return nil, err
		}

		addrs := w.host.Peerstore().Addrs(peerId)
		peers = append(peers, &Peer{
			ID:        peerId,
			Protocols: protocols,
			Connected: connected,
			Addrs:     addrs,
		})
	}
	return peers, nil
}

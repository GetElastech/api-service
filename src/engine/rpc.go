package engine

import (
	"net"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"

	"github.com/onflow/flow-go/engine"
	"github.com/onflow/flow-go/module"
	"github.com/onflow/flow-go/network"
	jsoncodec "github.com/onflow/flow-go/network/codec/json"
	"github.com/onflow/flow-go/utils/grpcutils"
	accessproto "github.com/onflow/flow/protobuf/go/flow/access"
)

// Config defines the configurable options for the gRPC server.
type Config struct {
	ListenAddr string
	MaxMsgSize int // In bytes
}

// RPC implements a gRPC server for the API service node
type RPC struct {
	unit   *engine.Unit
	log    zerolog.Logger
	server *grpc.Server // the gRPC server
	config Config
	me     module.Local
	codec  network.Codec

	//conduits map[network.Channel]network.Conduit
	// The proxy reads from the channel and returns it as GRPC stream to the client
	proxy accessproto.AccessAPIServer
}

// New returns a new RPC engine.
func New(log zerolog.Logger, me module.Local, config Config, proxy accessproto.AccessAPIServer) (*RPC, error) {

	log = log.With().Str("engine", "rpc").Logger()

	codec := jsoncodec.NewCodec()

	if config.MaxMsgSize == 0 {
		config.MaxMsgSize = grpcutils.DefaultMaxMsgSize
	}

	eng := &RPC{
		log:  log,
		unit: engine.NewUnit(),
		me:   me,
		server: grpc.NewServer(
			grpc.MaxRecvMsgSize(config.MaxMsgSize),
			grpc.MaxSendMsgSize(config.MaxMsgSize),
		),
		config: config,
		codec:  codec,
	}

	//conduitMap, err := registerConduits(net, state, eng)
	//if err != nil {
	//	return nil, fmt.Errorf("failed to initialize RPC: %w", err)
	//}
	//eng.conduits = conduitMap

	eng.proxy = proxy
	accessproto.RegisterAccessAPIServer(eng.server, proxy)

	return eng, nil
}

//// registerConduits registers for ALL channels and returns a map of engine id to conduit
//func registerConduits(net network.Network, state protocol.State, eng network.Engine) (map[network.Channel]network.Conduit, error) {
//
//	// create a list of all channels that don't change over time
//	channels := network.ChannelList{
//		engine.ConsensusCommittee,
//		engine.SyncCommittee,
//		engine.SyncExecution,
//		engine.PushTransactions,
//		engine.PushGuarantees,
//		engine.PushBlocks,
//		engine.PushReceipts,
//		engine.PushApprovals,
//		engine.RequestCollections,
//		engine.RequestChunks,
//	}
//
//	// add channels that are dependent on protocol state and change over time
//	// TODO need to update to register dynamic channels that are created on later epoch transitions
//	epoch := state.Final().Epochs().Current()
//
//	clusters, err := epoch.Clustering()
//	if err != nil {
//		return nil, fmt.Errorf("could not get clusters: %w", err)
//	}
//
//	for i := range clusters {
//		cluster, err := epoch.Cluster(uint(i))
//		if err != nil {
//			return nil, fmt.Errorf("could not get cluster: %w", err)
//		}
//		clusterID := cluster.RootBlock().Header.ChainID
//
//		// add the dynamic channels for the cluster
//		channels = append(
//			channels,
//			engine.ChannelConsensusCluster(clusterID),
//			engine.ChannelSyncCluster(clusterID),
//		)
//	}
//
//	conduitMap := make(map[network.Channel]network.Conduit, len(channels))
//
//	// Register for ALL channels here and return a map of conduits
//	for _, e := range channels {
//		c, err := net.Register(e, eng)
//		if err != nil {
//			return nil, fmt.Errorf("could not register collection provider engine: %w", err)
//		}
//		conduitMap[e] = c
//	}
//
//	return conduitMap, nil
//
//}

// Ready returns a ready channel that is closed once the engine has fully
// started. The RPC engine is ready when the gRPC server has successfully
// started.
func (e *RPC) Ready() <-chan struct{} {
	e.unit.Launch(e.serve)
	return e.unit.Ready()
}

// Done returns a done channel that is closed once the engine has fully stopped.
// It sends a signal to stop the gRPC server, then closes the channel.
func (e *RPC) Done() <-chan struct{} {
	return e.unit.Done(e.server.GracefulStop)
}

/*
// SubmitLocal submits an event originating on the local node.
func (e *RPC) SubmitLocal(event interface{}) {
	e.unit.Launch(func() {
		err := e.process(e.me.NodeID(), event)
		if err != nil {
			e.log.Error().Err(err).Msg("could not process submitted event")
		}
	})
}

// Submit submits the given event from the node with the given origin ID
// for processing in a non-blocking manner. It returns instantly and logs
// a potential processing error internally when done.
func (e *RPC) Submit(channel network.Channel, originID flow.Identifier, event interface{}) {
	e.unit.Launch(func() {
		err := e.process(originID, event)
		if err != nil {
			e.log.Error().Err(err).Msg("could not process submitted event")
		}
	})
}

// ProcessLocal processes an event originating on the local node.
func (e *RPC) ProcessLocal(event interface{}) error {
	return e.unit.Do(func() error {
		return e.process(e.me.NodeID(), event)
	})
}

// Process processes the given event from the node with the given origin ID in
// a blocking manner. It returns the potential processing error when done.
func (e *RPC) Process(channel network.Channel, originID flow.Identifier, event interface{}) error {
	return e.unit.Do(func() error {
		return e.process(originID, event)
	})
}

func (e *RPC) process(originID flow.Identifier, event interface{}) error {

	// json encode the message into bytes
	encodedMsg, err := e.codec.Encode(event)
	if err != nil {
		return fmt.Errorf("failed to encode message: %v", err)
	}

	// create a protobuf message
	flowMessage := ghost.FlowMessage{
		SenderID: originID[:],
		Message:  encodedMsg,
	}

	// write it to the channel
	select {
	case e.messages <- flowMessage:
	default:
		return fmt.Errorf("dropping message since queue is full: %v", err)
	}
	return nil
}
*/
// serve starts the gRPC server .
//
// When this function returns, the server is considered ready.
func (e *RPC) serve() {
	e.log.Info().Msgf("starting server on address %s", e.config.ListenAddr)

	l, err := net.Listen("tcp", e.config.ListenAddr)
	if err != nil {
		e.log.Err(err).Msg("failed to start server")
		return
	}

	err = e.server.Serve(l)
	if err != nil {
		e.log.Err(err).Msg("fatal error in server")
	}
}

package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/onflow/flow/protobuf/go/flow/access"

	"github.com/onflow/flow-go/engine/access/rpc/backend"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/utils/grpcutils"

	flowDpsAccess "github.com/GetElastech/flow-dps-access/api"
)

func NewFlowAPIService(protocolNodeAddressAndPort flow.IdentityList, executorNodeAddressAndPort flow.IdentityList, flowDpsHostUrl string, flowDpsListenPort string, flowDpsMaxCacheSize uint64, timeout time.Duration) (*FlowAPIService, error) {
	protocolClients := make([]access.AccessAPIClient, protocolNodeAddressAndPort.Count())
	for i, identity := range protocolNodeAddressAndPort {
		identity.NetworkPubKey = nil
		if identity.NetworkPubKey == nil {
			clientRPCConnection, err := grpc.Dial(
				identity.Address,
				grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(grpcutils.DefaultMaxMsgSize)),
				grpc.WithInsecure(), //nolint:staticcheck
				backend.WithClientUnaryInterceptor(timeout))
			if err != nil {
				return nil, err
			}

			protocolClients[i] = access.NewAccessAPIClient(clientRPCConnection)
		} else {
			tlsConfig, err := grpcutils.DefaultClientTLSConfig(identity.NetworkPubKey)
			if err != nil {
				return nil, err
			}

			clientRPCConnection, err := grpc.Dial(
				identity.Address,
				grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(grpcutils.DefaultMaxMsgSize)),
				grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
				backend.WithClientUnaryInterceptor(timeout))
			if err != nil {
				return nil, err
			}

			protocolClients[i] = access.NewAccessAPIClient(clientRPCConnection)
		}
	}

	executorClients := make([]access.AccessAPIClient, executorNodeAddressAndPort.Count())
	for i, identity := range executorNodeAddressAndPort {
		identity.NetworkPubKey = nil
		if identity.NetworkPubKey == nil {
			clientRPCConnection, err := grpc.Dial(
				identity.Address,
				grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(grpcutils.DefaultMaxMsgSize)),
				grpc.WithInsecure(), //nolint:staticcheck
				backend.WithClientUnaryInterceptor(timeout))
			if err != nil {
				return nil, err
			}

			executorClients[i] = access.NewAccessAPIClient(clientRPCConnection)
		} else {
			tlsConfig, err := grpcutils.DefaultClientTLSConfig(identity.NetworkPubKey)
			if err != nil {
				return nil, err
			}

			clientRPCConnection, err := grpc.Dial(
				identity.Address,
				grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(grpcutils.DefaultMaxMsgSize)),
				grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
				backend.WithClientUnaryInterceptor(timeout))
			if err != nil {
				return nil, err
			}

			executorClients[i] = access.NewAccessAPIClient(clientRPCConnection)
		}
	}

	flowDpsAccessServer, listener, err := NewDpsAccessServer(flowDpsHostUrl, flowDpsListenPort, flowDpsMaxCacheSize)
	if err != nil {
		return nil, err
	}

	ret := &FlowAPIService{
		flowDpsAccess:     flowDpsAccessServer,
		upstreamProtocol:  protocolClients,
		upstreamExecution: executorClients,
		roundRobin:        0,
		lock:              sync.Mutex{},
		dpsListener:       listener,
	}
	return ret, nil
}

// BootstrapIdentities converts the bootstrap node addresses and keys to a Flow Identity list where
// each Flow Identity is initialized with the passed address, the networking key
// and the Node ID set to ZeroID, role set to Access, 0 stake and no staking key.
func BootstrapIdentities(addresses []string, keys []string) (flow.IdentityList, error) {
	if len(addresses) != len(keys) {
		return nil, fmt.Errorf("number of addresses and keys provided for the boostrap nodes don't match")
	}

	ids := make([]*flow.Identity, len(addresses))
	for i, address := range addresses {
		key := keys[i]

		// create the identity of the peer by setting only the relevant fields
		ids[i] = &flow.Identity{
			NodeID:        flow.ZeroID, // the NodeID is the hash of the staking key and for the public network it does not apply
			Address:       address,
			Role:          flow.RoleAccess, // the upstream node has to be an access node
			NetworkPubKey: nil,
		}

		// json unmarshaller needs a quotes before and after the string
		// the pflags.StringSliceVar does not retain quotes for the command line arg even if escaped with \"
		// hence this additional check to ensure the key is indeed quoted
		if !strings.HasPrefix(key, "\"") {
			key = fmt.Sprintf("\"%s\"", key)
		}
		// networking public key
		_ = json.Unmarshal([]byte(key), &ids[i].NetworkPubKey)
	}
	return ids, nil
}

type FlowAPIService struct {
	access.AccessAPIServer
	flowDpsAccess     *flowDpsAccess.Server
	lock              sync.Mutex
	roundRobin        int
	upstreamProtocol  []access.AccessAPIClient
	upstreamExecution []access.AccessAPIClient
	dpsListener       net.Listener
}

func (h *FlowAPIService) SetLocalAPI(local access.AccessAPIServer) {
	h.AccessAPIServer = local
}

func (h *FlowAPIService) clientProtocol() (access.AccessAPIClient, error) {
	if h.upstreamProtocol == nil || len(h.upstreamProtocol) == 0 {
		return nil, status.Errorf(codes.Unimplemented, "method not implemented")
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	h.roundRobin++
	h.roundRobin = h.roundRobin % len(h.upstreamProtocol)
	ret := h.upstreamProtocol[h.roundRobin]

	return ret, nil
}

func (h *FlowAPIService) clientExecution() (access.AccessAPIClient, error) {
	if h.upstreamExecution == nil || len(h.upstreamExecution) == 0 {
		return nil, status.Errorf(codes.Unimplemented, "method not implemented")
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	h.roundRobin++
	h.roundRobin = h.roundRobin % len(h.upstreamExecution)
	ret := h.upstreamExecution[h.roundRobin]

	return ret, nil
}

func (h *FlowAPIService) Ping(context context.Context, req *access.PingRequest) (*access.PingResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientProtocol()
	if err != nil {
		return nil, err
	}
	return upstream.Ping(context, req)
}

func (h *FlowAPIService) GetLatestBlockHeader(context context.Context, req *access.GetLatestBlockHeaderRequest) (*access.BlockHeaderResponse, error) {
	// This is a passthrough request
	return h.flowDpsAccess.GetLatestBlockHeader(context, req)
}

func (h *FlowAPIService) GetBlockHeaderByID(context context.Context, req *access.GetBlockHeaderByIDRequest) (*access.BlockHeaderResponse, error) {
	// This is a passthrough request
	return h.flowDpsAccess.GetBlockHeaderByID(context, req)
}

func (h *FlowAPIService) GetBlockHeaderByHeight(context context.Context, req *access.GetBlockHeaderByHeightRequest) (*access.BlockHeaderResponse, error) {
	// This is a passthrough request
	return h.flowDpsAccess.GetBlockHeaderByHeight(context, req)
}

func (h *FlowAPIService) GetLatestBlock(context context.Context, req *access.GetLatestBlockRequest) (*access.BlockResponse, error) {
	// This is a passthrough request
	return h.flowDpsAccess.GetLatestBlock(context, req)
}

func (h *FlowAPIService) GetBlockByID(context context.Context, req *access.GetBlockByIDRequest) (*access.BlockResponse, error) {
	// This is a passthrough request
	return h.flowDpsAccess.GetBlockByID(context, req)
}

func (h *FlowAPIService) GetBlockByHeight(context context.Context, req *access.GetBlockByHeightRequest) (*access.BlockResponse, error) {
	// This is a passthrough request
	return h.flowDpsAccess.GetBlockByHeight(context, req)
}

func (h *FlowAPIService) GetCollectionByID(context context.Context, req *access.GetCollectionByIDRequest) (*access.CollectionResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientProtocol()
	if err != nil {
		return nil, err
	}
	return upstream.GetCollectionByID(context, req)
}

func (h *FlowAPIService) SendTransaction(context context.Context, req *access.SendTransactionRequest) (*access.SendTransactionResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientProtocol()
	if err != nil {
		return nil, err
	}
	return upstream.SendTransaction(context, req)
}

func (h *FlowAPIService) GetTransaction(context context.Context, req *access.GetTransactionRequest) (*access.TransactionResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.GetTransaction(context, req)
}

func (h *FlowAPIService) GetTransactionResult(context context.Context, req *access.GetTransactionRequest) (*access.TransactionResultResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.GetTransactionResult(context, req)
}

func (h *FlowAPIService) GetTransactionResultByIndex(context context.Context, req *access.GetTransactionByIndexRequest) (*access.TransactionResultResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.GetTransactionResultByIndex(context, req)
}

func (h *FlowAPIService) GetAccount(context context.Context, req *access.GetAccountRequest) (*access.GetAccountResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.GetAccount(context, req)
}

func (h *FlowAPIService) GetAccountAtLatestBlock(context context.Context, req *access.GetAccountAtLatestBlockRequest) (*access.AccountResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.GetAccountAtLatestBlock(context, req)
}

func (h *FlowAPIService) GetAccountAtBlockHeight(context context.Context, req *access.GetAccountAtBlockHeightRequest) (*access.AccountResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.GetAccountAtBlockHeight(context, req)
}

func (h *FlowAPIService) ExecuteScriptAtLatestBlock(context context.Context, req *access.ExecuteScriptAtLatestBlockRequest) (*access.ExecuteScriptResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.ExecuteScriptAtLatestBlock(context, req)
}

func (h *FlowAPIService) ExecuteScriptAtBlockID(context context.Context, req *access.ExecuteScriptAtBlockIDRequest) (*access.ExecuteScriptResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.ExecuteScriptAtBlockID(context, req)
}

func (h *FlowAPIService) ExecuteScriptAtBlockHeight(context context.Context, req *access.ExecuteScriptAtBlockHeightRequest) (*access.ExecuteScriptResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.ExecuteScriptAtBlockHeight(context, req)
}

func (h *FlowAPIService) GetEventsForHeightRange(context context.Context, req *access.GetEventsForHeightRangeRequest) (*access.EventsResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.GetEventsForHeightRange(context, req)
}

func (h *FlowAPIService) GetEventsForBlockIDs(context context.Context, req *access.GetEventsForBlockIDsRequest) (*access.EventsResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.GetEventsForBlockIDs(context, req)
}

func (h *FlowAPIService) GetNetworkParameters(context context.Context, req *access.GetNetworkParametersRequest) (*access.GetNetworkParametersResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.GetNetworkParameters(context, req)
}

func (h *FlowAPIService) GetLatestProtocolStateSnapshot(context context.Context, req *access.GetLatestProtocolStateSnapshotRequest) (*access.ProtocolStateSnapshotResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientProtocol()
	if err != nil {
		return nil, err
	}
	return upstream.GetLatestProtocolStateSnapshot(context, req)
}

func (h *FlowAPIService) GetExecutionResultForBlockID(context context.Context, req *access.GetExecutionResultForBlockIDRequest) (*access.ExecutionResultForBlockIDResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.GetExecutionResultForBlockID(context, req)
}

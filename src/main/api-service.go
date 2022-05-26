package main

import (
	engine2 "github.com/onflow/api-service/m/v2/engine"
	"github.com/onflow/api-service/m/v2/proxy"
	"github.com/spf13/pflag"
	"time"

	"github.com/onflow/flow-go/cmd"
	"github.com/onflow/flow-go/module"
	"github.com/onflow/flow-go/network"
	"github.com/onflow/flow-go/network/validator"
)

func main() {
	var (
		rpcConf engine2.Config
	)

	var apiTimeout time.Duration
	var upstreamNodeAddresses []string
	var upstreamNodePublicKeys []string

	nodeBuilder := cmd.FlowNode("api-service")
	nodeBuilder.ExtraFlags(func(flags *pflag.FlagSet) {
		flags.StringVarP(&rpcConf.ListenAddr, "rpc-addr", "r", "localhost:9000", "the address the GRPC server listens on")
		flags.DurationVar(&apiTimeout, "flow-api-timeout", 3*time.Second, "tcp timeout for Flow API gRPC socket")
		flags.StringSliceVar(&upstreamNodeAddresses, "upstream-node-addresses", upstreamNodeAddresses, "the network addresses of the bootstrap access node if this is an observer e.g. access-001.mainnet.flow.org:9653,access-002.mainnet.flow.org:9653")
		flags.StringSliceVar(&upstreamNodePublicKeys, "upstream-node-public-keys", upstreamNodePublicKeys, "the networking public key of the bootstrap access node if this is an observer (in the same order as the bootstrap node addresses) e.g. \"d57a5e9c5.....\",\"44ded42d....\"")
	})

	if err := nodeBuilder.Initialize(); err != nil {
		nodeBuilder.Logger.Fatal().Err(err).Send()
	}

	nodeBuilder.
		Module("message validators", func(node *cmd.NodeConfig) error {
			validators := []network.MessageValidator{
				// filter out messages sent by this node itself
				validator.ValidateNotSender(node.Me.NodeID()),
				// but retain all the 1-k messages even if they are not intended for this node
			}
			node.MsgValidators = validators
			return nil
		}).
		Component("RPC engine", func(node *cmd.NodeConfig) (module.ReadyDoneAware, error) {
			ids, err := proxy.BootstrapIdentities(upstreamNodeAddresses, upstreamNodePublicKeys)
			if err != nil {
				return nil, err
			}
			api, err := proxy.NewFlowAPIService(ids, apiTimeout)
			if err != nil {
				return nil, err
			}

			rpcEng, err := engine2.New(node.Network, node.Logger, node.Me, node.State, rpcConf, api)
			return rpcEng, err
		})

	node, err := nodeBuilder.Build()
	if err != nil {
		nodeBuilder.Logger.Fatal().Err(err).Send()
	}
	node.Run()
}

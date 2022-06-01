package main

import (
	"github.com/onflow/api-service/m/v2/cmd/api-service/builder"
	"github.com/onflow/api-service/m/v2/cmd/engine"
	"github.com/onflow/api-service/m/v2/cmd/proxy"
	"github.com/onflow/api-service/m/v2/cmd/service"
	"github.com/onflow/flow/protobuf/go/flow/access"
	"github.com/spf13/pflag"
	"time"
)

func main() {
	var (
		rpcConf                engine.Config
		apiTimeout             time.Duration
		upstreamNodeAddresses  []string
		upstreamNodePublicKeys []string
		api                    access.AccessAPIServer
	)

	nodeBuilder := builder.NewFlowAPIServiceBuilder()

	nodeBuilder.ExtraFlags(func(flags *pflag.FlagSet) {
		flags.StringVarP(&rpcConf.ListenAddr, "rpc-addr", "r", ":9000", "the address the GRPC server listens on")
		flags.DurationVar(&apiTimeout, "flow-api-timeout", 3*time.Second, "tcp timeout for Flow API gRPC socket")
		flags.StringSliceVar(&upstreamNodeAddresses, "upstream-node-addresses", []string{}, "the network addresses of the bootstrap access node if this is an observer e.g. access-001.mainnet.flow.org:9653,access-002.mainnet.flow.org:9653")
		flags.StringSliceVar(&upstreamNodePublicKeys, "upstream-node-public-keys", []string{}, "the networking public key of the bootstrap access node if this is an observer (in the same order as the bootstrap node addresses) e.g. \"d57a5e9c5.....\",\"44ded42d....\"")
	})

	if err := nodeBuilder.Initialize(); err != nil {
		nodeBuilder.ServiceConfig.Logger.Fatal().Err(err).Send()
	}

	nodeBuilder.
		Module("API Service", func(node *service.ServiceConfig) error {
			ids, err := proxy.BootstrapIdentities(upstreamNodeAddresses, upstreamNodePublicKeys)
			if err != nil {
				nodeBuilder.ServiceConfig.Logger.Info().Err(err)
				return err
			}
			api, err = proxy.NewFlowAPIService(ids, apiTimeout)
			if err != nil {
				nodeBuilder.ServiceConfig.Logger.Info().Err(err)
				return err
			}
			nodeBuilder.ServiceConfig.Logger.Info().Msg("API Service started")
			return nil
		}).
		Module("RPC engine", func(node *service.ServiceConfig) error {
			rpcEng, err := engine.New(node.Logger, rpcConf, api)
			if err != nil {
				nodeBuilder.ServiceConfig.Logger.Info().Err(err)
				return err
			}
			nodeBuilder.RpcEngine = rpcEng
			nodeBuilder.ServiceConfig.Logger.Info().Str("module", node.Name).Msg("RPC engine started")
			return nil
		})

	node, err := nodeBuilder.Build()
	if err != nil {
		nodeBuilder.ServiceConfig.Logger.Err(err)
	}
	node.Run()
}

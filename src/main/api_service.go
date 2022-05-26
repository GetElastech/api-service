package main

import (
	"github.com/onflow/api-service/m/v2/proxy"
	"github.com/onflow/api-service/m/v2/proxy/builder"
	"github.com/onflow/api-service/m/v2/proxy/engine"
	"github.com/onflow/api-service/m/v2/service"
	"github.com/onflow/flow/protobuf/go/flow/access"
	"github.com/spf13/pflag"
	"os"
	"time"
)

func main() {
	var (
		rpcConf    engine.Config
		apiTimeout time.Duration
		upstreamNodeAddresses  []string
		upstreamNodePublicKeys []string
		api        access.AccessAPIServer
		rpc        *engine.RPC
	)

	nodeBuilder := builder.NewFlowAPIServiceBuilder()

	nodeBuilder.ExtraFlags(func(flags *pflag.FlagSet) {
		flags.StringVarP(&rpcConf.ListenAddr, "rpc-addr", "r", "localhost:9000", "the address the GRPC server listens on")
		flags.DurationVar(&apiTimeout, "flow-api-timeout", 3*time.Second, "tcp timeout for Flow API gRPC socket")
		flags.StringSliceVar(&upstreamNodeAddresses, "upstream-node-addresses", upstreamNodeAddresses, "the network addresses of the bootstrap access node if this is an observer e.g. access-001.mainnet.flow.org:9653,access-002.mainnet.flow.org:9653")
		flags.StringSliceVar(&upstreamNodePublicKeys, "upstream-node-public-keys", upstreamNodePublicKeys, "the networking public key of the bootstrap access node if this is an observer (in the same order as the bootstrap node addresses) e.g. \"d57a5e9c5.....\",\"44ded42d....\"")
	})

	if err := nodeBuilder.Initialize(); err != nil {
		nodeBuilder.ServiceConfig.Logger.Fatal().Err(err).Send()
	}

	nodeBuilder.
		Module("API Service", func(node *service.ServiceConfig) error {
			ids, err := proxy.BootstrapIdentities(upstreamNodeAddresses, upstreamNodePublicKeys)
			if err != nil {
				return err
			}
			api, err = proxy.NewFlowAPIService(ids, apiTimeout)
			if err != nil {
				return err
			}
			return err
		}).
		Module("RPC engine", func(node *service.ServiceConfig) error {
			rpcEng, err := engine.New(node.Logger, rpcConf, api)
			if err != nil {
				return err
			}
			rpc = rpcEng
			return err
		})

	fs, err := nodeBuilder.Build()
	if err != nil {
		nodeBuilder.ServiceConfig.Logger.Err(err)
		os.Exit(2)
	}

	(*fs).Run()
}

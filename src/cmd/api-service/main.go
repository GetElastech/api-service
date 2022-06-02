package main

import (
	"fmt"
	"github.com/onflow/api-service/m/v2/cmd/api-service/builder"
	"github.com/onflow/api-service/m/v2/cmd/engine"
	"github.com/onflow/api-service/m/v2/cmd/proxy"
	"github.com/onflow/api-service/m/v2/cmd/service"
	"github.com/onflow/flow/protobuf/go/flow/access"
	"os"
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

	serviceBuilder := builder.NewFlowAPIServiceBuilder()

	flags := &serviceBuilder.ServiceConfig.Flags
	flags.StringVarP(&rpcConf.ListenAddr, "rpc-addr", "r", ":9000", "the address the GRPC server listens on")
	flags.DurationVar(&apiTimeout, "flow-api-timeout", 3*time.Second, "tcp timeout for Flow API gRPC socket")
	flags.StringSliceVar(&upstreamNodeAddresses, "upstream-node-addresses", []string{"127.0.0.1:3569"}, "the network addresses of the bootstrap access node if this is an observer e.g. access-001.mainnet.flow.org:9653,access-002.mainnet.flow.org:9653")
	flags.StringSliceVar(&upstreamNodePublicKeys, "upstream-node-public-keys", []string{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, "the networking public key of the bootstrap access node if this is an observer (in the same order as the bootstrap node addresses) e.g. \"d57a5e9c5.....\",\"44ded42d....\"")

	err := flags.Parse(os.Args[1:])
	if err != nil {
		serviceBuilder.ServiceConfig.Logger.Fatal().Err(err)
	}
	serviceBuilder.ServiceConfig.Logger.Info().
		Str("upstream-node-addresses", fmt.Sprintf("%v", upstreamNodeAddresses)).
		Str("upstream-node-public-keys", fmt.Sprintf("%v", upstreamNodePublicKeys))
	// print all flags
	serviceBuilder.ServiceConfig.Logger.Info().Str("upstream-node-addresses", fmt.Sprintf("%v", upstreamNodeAddresses)).Msg("flags loaded")

	if err := serviceBuilder.Initialize(); err != nil {
		serviceBuilder.ServiceConfig.Logger.Fatal().Err(err).Send()
	}

	serviceBuilder.
		Module("API Service", func(node *service.ServiceConfig) error {
			ids, err := proxy.BootstrapIdentities(upstreamNodeAddresses, upstreamNodePublicKeys)
			if err != nil {
				serviceBuilder.ServiceConfig.Logger.Info().Err(err)
				return err
			}
			for _, id := range ids {
				serviceBuilder.ServiceConfig.Logger.Info().Str("upstream", id.Address).Msg("API Service client")
			}
			api, err = proxy.NewFlowAPIService(ids, apiTimeout)
			if err != nil {
				serviceBuilder.ServiceConfig.Logger.Info().Err(err)
				return err
			}
			serviceBuilder.ServiceConfig.Logger.Info().Str("cmd", fmt.Sprintf("%v", upstreamNodeAddresses)).Msg("API Service started")
			return nil
		}).
		Module("RPC engine", func(node *service.ServiceConfig) error {
			rpcEng, err := engine.New(node.Logger, rpcConf, api)
			if err != nil {
				serviceBuilder.ServiceConfig.Logger.Info().Err(err)
				return err
			}
			serviceBuilder.RpcEngine = rpcEng
			serviceBuilder.ServiceConfig.Logger.Info().Str("module", node.Name).Msg("RPC engine started")
			return nil
		}).
		Component("Start listening", func(node *service.ServiceConfig) error {
			// run all modules
			<-serviceBuilder.RpcEngine.Ready()
			serviceBuilder.ServiceConfig.Logger.Info().Msg("Flow API Service Ready")
			return nil
		})

	service, err := serviceBuilder.Build()
	if err != nil {
		serviceBuilder.ServiceConfig.Logger.Err(err)
	}
	err = service.Run()
	if err != nil {
		serviceBuilder.ServiceConfig.Logger.Err(err)
	}
}

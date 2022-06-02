package builder

import (
	"github.com/onflow/api-service/m/v2/cmd/engine"
	"github.com/onflow/api-service/m/v2/cmd/service"
	"os"
	"os/signal"
	"syscall"
)

type FlowAPIServiceCmd struct {
	service.FlowService
	ServiceConfig service.ServiceConfig
	RpcEngine     *engine.RPC
}

type FlowAPIServiceBuilder struct {
	*service.FlowServiceBuilder
	RpcEngine *engine.RPC
}

func (fsb *FlowAPIServiceBuilder) Initialize() error {
	return nil
}

func (fsb *FlowAPIServiceBuilder) Build() (*FlowAPIServiceCmd, error) {
	fs, err := fsb.FlowServiceBuilder.Build()
	if err != nil {
		return nil, err
	}
	return &FlowAPIServiceCmd{
		FlowService:   *fs,
		ServiceConfig: fsb.ServiceConfig,
		RpcEngine:     fsb.RpcEngine,
	}, nil
}

func (fsb *FlowAPIServiceCmd) Run() error {
	// 1: Start up
	// Start all the components
	err := fsb.ServiceConfig.Start()
	if err != nil {
		return err
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, syscall.SIGINT)
	defer signal.Stop(sigint)
	<-sigint

	fsb.ServiceConfig.Logger.Info().Msg("Flow API Service Done")
	<-fsb.RpcEngine.Done()

	return nil
}

func NewFlowAPIServiceBuilder() *FlowAPIServiceBuilder {
	return &FlowAPIServiceBuilder{
		FlowServiceBuilder: service.NewFlowServiceBuilder("api-service"),
	}
}

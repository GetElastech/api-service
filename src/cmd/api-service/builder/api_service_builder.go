package builder

import (
	"github.com/onflow/api-service/m/v2/cmd/engine"
	"github.com/onflow/api-service/m/v2/cmd/service"
	"time"
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

func (fsb *FlowAPIServiceCmd) Run() {
	// run all modules
	fsb.RpcEngine.Ready()
	fsb.ServiceConfig.Logger.Info().Msg("Flow API Service Ready")
	time.Sleep(100 * time.Second)
}

func NewFlowAPIServiceBuilder() *FlowAPIServiceBuilder {
	return &FlowAPIServiceBuilder{
		FlowServiceBuilder: service.NewFlowServiceBuilder("api-service"),
	}
}

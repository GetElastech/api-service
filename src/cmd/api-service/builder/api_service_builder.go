package builder

import (
	"github.com/onflow/api-service/m/v2/cmd/engine"
	"github.com/onflow/api-service/m/v2/cmd/service"
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
	// start all components
	err := fsb.ServiceConfig.Start()
	if err != nil {
		return err
	}
	return nil
}

func NewFlowAPIServiceBuilder() *FlowAPIServiceBuilder {
	return &FlowAPIServiceBuilder{
		FlowServiceBuilder: service.NewFlowServiceBuilder("api-service"),
	}
}

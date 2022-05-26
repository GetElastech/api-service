package builder

import (
	"github.com/onflow/api-service/m/v2/cmd/service"
	"time"
)

type FlowAPIServiceCmd struct {
	service.FlowService
	ServiceConfig service.ServiceConfig
}

type FlowAPIServiceBuilder struct {
	*service.FlowServiceBuilder
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
	}, nil
}

func (fsb *FlowAPIServiceCmd) Run() {
	time.Sleep(100 * time.Second)
}

func NewFlowAPIServiceBuilder() *FlowAPIServiceBuilder {
	return &FlowAPIServiceBuilder{
		FlowServiceBuilder: service.NewFlowServiceBuilder("api-service"),
	}
}

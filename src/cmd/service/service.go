package service

import (
	"github.com/onflow/flow-go/module/irrecoverable"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"os"
)

func NewFlowServiceBuilder(name string) *FlowServiceBuilder {
	return &FlowServiceBuilder{
		ServiceConfig: ServiceConfig{
			Name:   name,
			Logger: zerolog.New(os.Stderr),
		},
	}
}

type ServiceConfig struct {
	Name   string
	Logger zerolog.Logger
	flags  pflag.FlagSet
}

type FlowService interface {
	Run()
}

type FlowServiceBuilder struct {
	FlowService
	ServiceConfig ServiceConfig
	modules       []namedModuleFunc
}

func (fsb *FlowServiceBuilder) Build() (*FlowService, error) {
	// build all modules
	for _, f := range fsb.modules {
		if err := f.fn(&fsb.ServiceConfig); err != nil {
			fsb.ServiceConfig.Logger.Err(err)
		}
		fsb.ServiceConfig.Logger.Info().Str("module", f.name).Msg("service module started")
	}
	return &fsb.FlowService, nil
}

func (fsb *FlowServiceBuilder) Start(irrecoverable.SignalerContext) {
}

func (receiver *FlowServiceBuilder) Run() {
}

// ExtraFlags enables binding additional flags beyond those defined in BaseConfig.
func (fnb *FlowServiceBuilder) ExtraFlags(f func(*pflag.FlagSet)) *FlowServiceBuilder {
	f(&fnb.ServiceConfig.flags)
	return fnb
}

type BuilderFunc func(serviceConfig *ServiceConfig) error

type namedModuleFunc struct {
	fn   BuilderFunc
	name string
}

// Module enables setting up dependencies of the engine with the builder context.
func (fsb *FlowServiceBuilder) Module(name string, f BuilderFunc) *FlowServiceBuilder {
	fsb.modules = append(fsb.modules, namedModuleFunc{
		fn:   f,
		name: name,
	})
	return fsb
}

func (fnb *FlowServiceBuilder) onStart() error {

	// seed random generator
	//rand.Seed(time.Now().UnixNano())
	//
	//// init nodeinfo by reading the private bootstrap file if not already set
	//if fnb.NodeID == flow.ZeroID {
	//	fnb.initNodeInfo()
	//}

	// run all modules
	//for _, f := range fnb.*fnb.FlowNodeBuilder.modules {
	//	if err := fnb.handleModule(f); err != nil {
	//		return err
	//	}
	//}
	//
	//// run all components
	//return fnb.handleComponents()

	return nil
}

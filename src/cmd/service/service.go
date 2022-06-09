package service

import (
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

type FlowService interface {
	Run()
}

type ServiceConfig struct {
	Name       string
	Logger     zerolog.Logger
	Flags      pflag.FlagSet
	Components []namedModuleFunc
}

type FlowServiceBuilder struct {
	FlowService
	ServiceConfig ServiceConfig
	modules       []namedModuleFunc
}

func (fsb *FlowServiceBuilder) ParseAndPrintFlags() error {
	// parse configuration parameters
	pflag.Parse()

	// print all flags
	log := fsb.ServiceConfig.Logger.Info()

	pflag.VisitAll(func(flag *pflag.Flag) {
		log = log.Str(flag.Name, flag.Value.String())
	})

	fsb.ServiceConfig.Logger.Info().Msg("flags loaded")
	return nil
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

//// ExtraFlags enables binding additional Flags beyond those defined in BaseConfig.
//func (fnb *FlowServiceBuilder) ExtraFlags(f func(*pflag.FlagSet)) *FlowServiceBuilder {
//	f(&fnb.ServiceConfig.Flags)
//	return fnb
//}

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

// Component adds a new component to the node that conforms to the ReadyDoneAware
// interface, and throws a Fatal() when an irrecoverable error is encountered.
func (fsb *FlowServiceBuilder) Component(name string, f BuilderFunc) *FlowServiceBuilder {
	fsb.ServiceConfig.Components = append(fsb.ServiceConfig.Components, namedModuleFunc{
		fn:   f,
		name: name,
	})
	return fsb
}

func (fsc *ServiceConfig) Start() error {
	// start all components
	for _, f := range fsc.Components {
		if err := f.fn(fsc); err != nil {
			fsc.Logger.Error().Str("module", f.name).Err(err)
			return err
		}
		fsc.Logger.Info().Str("module", f.name).Msg("Service Component Started")
	}

	return nil
}

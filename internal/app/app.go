package app

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

const primaryEngine = "primary"

func New(ctx context.Context, dsn string) (*system.System, error) {
	projection, typeDecoder := calls.Projection()

	engine := system.EngineConfig{Driver: "memory"}
	if dsn != "" {
		engine = system.EngineConfig{Driver: "sqlite", DSN: dsn}
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{primaryEngine: engine},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: primaryEngine},
			{Role: system.RoleProjections, Engine: primaryEngine},
		},
	}

	domain := system.DomainConfig{
		Events: []event.Type{calls.EventObserved, calls.EventCorrected, calls.EventEnriched},
		Commands: func(sys *system.System) {
			mustRegister(system.RegisterDecider(sys, calls.StreamType, decider.Decider[calls.CallState]{
				Initial: calls.CallState{},
				Apply:   calls.Apply,
			}))

			mustRegister(system.RegisterCommand[calls.RecordCallCmd, calls.CallState](sys, calls.CommandRecord,
				func(cmdCtx context.Context, cmd calls.RecordCallCmd) system.Op[calls.CallState] {
					return system.Execute(
						cmdCtx,
						cmd.StreamID(),
						calls.StreamType,
						calls.Record(cmd.Observation, cmd.StreamID()),
					)
				}))

			mustRegister(system.RegisterCommand[calls.EnrichCallCmd, calls.CallState](sys, calls.CommandEnrich,
				func(cmdCtx context.Context, cmd calls.EnrichCallCmd) system.Op[calls.CallState] {
					return system.Execute(
						cmdCtx,
						cmd.StreamID(),
						calls.StreamType,
						calls.Enrich(cmd.Observation, cmd.Enrichment, cmd.StreamID()),
					)
				}))
		},
		Projections:           projection,
		ProjectionTypeDecoder: typeDecoder,
	}

	return system.New(ctx, domain, deployment)
}

func mustRegister(err error) {
	if err != nil {
		panic(err)
	}
}

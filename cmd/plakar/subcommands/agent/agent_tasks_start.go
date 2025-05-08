package agent

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/scheduler"
)

type AgentTasksStart struct {
	subcommands.SubcommandBase
}

func (cmd *AgentTasksStart) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("agent tasks start", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}
	flags.Parse(args)

	return nil
}

func (cmd *AgentTasksStart) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if agentContextSingleton == nil {
		return 1, fmt.Errorf("agent not started")
	}

	if agentContextSingleton.schedulerConfig == nil {
		return 1, fmt.Errorf("agent scheduler does not have a configuration")
	}

	if agentContextSingleton.schedulerState&AGENT_SCHEDULER_RUNNING != 0 {
		return 1, fmt.Errorf("agent scheduler already running")
	}

	// this needs to execute in the agent context, not the client context
	agentContextSingleton.schedulerCtx = appcontext.NewAppContextFrom(agentContextSingleton.agentCtx)
	go func() {
		scheduler.NewScheduler(agentContextSingleton.schedulerCtx, agentContextSingleton.schedulerConfig).Run()
	}()

	agentContextSingleton.schedulerState = AGENT_SCHEDULER_RUNNING
	return 0, nil
}

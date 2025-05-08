package agent

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/repository"
)

type AgentTasksStop struct {
	subcommands.SubcommandBase
}

func (cmd *AgentTasksStop) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("agent tasks stop", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}
	flags.Parse(args)

	return nil
}

func (cmd *AgentTasksStop) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if agentContextSingleton == nil {
		return 1, fmt.Errorf("agent not started")
	}

	if agentContextSingleton.schedulerState&AGENT_SCHEDULER_RUNNING == 0 {
		return 1, fmt.Errorf("agent scheduler not running")
	}

	fmt.Fprintf(ctx.Stderr, "Stopping agent scheduler... (may take some time)\n")
	agentContextSingleton.schedulerCtx.Cancel()
	agentContextSingleton.schedulerState = AGENT_SCHEDULER_STOPPED
	fmt.Fprintf(ctx.Stderr, "done !\n")
	agentContextSingleton.schedulerCtx = nil
	return 0, nil
}

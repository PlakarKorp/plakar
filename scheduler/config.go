package scheduler

import (
	"fmt"
	"io"
	"sort"
)

type Configuration struct {
	Jobs map[string]*Job
}

func (cfg *Configuration) Write(out io.Writer) {
	var names []string
	for name := range cfg.Jobs {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		job := cfg.Jobs[name]
		fmt.Fprintf(out, "job %q\n", name)
		fmt.Fprintln(out, "  ", job.Task.String())
		for _, sched := range job.Schedules {
			fmt.Fprintln(out, "    ", sched.String())
		}
	}
}

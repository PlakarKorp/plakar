package scheduler

import (
	"bytes"
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

func ParseConfigBytes(configBytes []byte) (*Configuration, error) {
	file := viper.New()
	file.SetConfigType("yaml")

	if err := file.ReadConfig(bytes.NewReader(configBytes)); err != nil {
		return nil, fmt.Errorf("failed to read configuration data: %w", err)
	}

	var config Configuration

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result: &config,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			BackupConfigCheckDecodeHook(),
			SyncDirectionDecodeHook(),
			DurationDecodeHook(),
		),
		ErrorUnused: true, // errors out if there are extra/unmapped keys
	})
	if err != nil {
		return nil, fmt.Errorf("creating decoder: %w", err)
	}

	if err := decoder.Decode(file.AllSettings()); err != nil {
		return nil, fmt.Errorf("decoding config: %w", err)
	}

	// Set default values for SyncConfig.Direction.
	for i := range config.Agent.Tasks {
		for j := range config.Agent.Tasks[i].Sync {
			if config.Agent.Tasks[i].Sync[j].Direction == "" {
				config.Agent.Tasks[i].Sync[j].Direction = SyncDirectionTo
			}
		}
	}

	validate := validator.New(validator.WithRequiredStructEnabled())

	validate.RegisterStructValidation(func(sl validator.StructLevel) {
		obj := sl.Current().Interface().(Task)
		if obj.Backup == nil && len(obj.Check) == 0 && len(obj.Restore) == 0 && len(obj.Sync) == 0 {
			sl.ReportError(obj, "Task", "Task", "atleastone", "at least one of Backup, Check, Restore, or Sync must be set")
		}
	}, Task{})

	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &config, nil
}

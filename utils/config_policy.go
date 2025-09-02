package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/PlakarKorp/kloset/locate"
	"go.yaml.in/yaml/v3"
)

type policiesConfig struct {
	Version  string                           `yaml:"version"`
	Policies map[string]*locate.LocateOptions `yaml:"policies"`
}

func (c *policiesConfig) Has(name string) bool {
	_, ok := c.Policies[name]
	return ok
}

func (c *policiesConfig) Add(name string) {
	c.Policies[name] = &locate.LocateOptions{}
}

func (c *policiesConfig) setInt(value string, p *int) error {
	i, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("invalid value: %w", err)
	}
	if i < 0 {
		return fmt.Errorf("negative value")
	}
	*p = i
	return nil
}

func (c *policiesConfig) setTime(value string, p *time.Time) error {
	t, err := locate.ParseTimeFlag(value)
	if err != nil {
		return fmt.Errorf("invalid value: %w", err)
	}
	*p = t
	return nil
}

func (c *policiesConfig) setStringList(value string, p *[]string) error {
	*p = strings.Split(value, ",")
	return nil
}

func (c *policiesConfig) locateField(name string, key string) (any, error) {
	p, ok := c.Policies[name]
	if !ok {
		return nil, fmt.Errorf("entry not found")
	}

	switch key {
	case "before":
		return &p.Filters.Before, nil
	case "since":
		return &p.Filters.Since, nil
	case "name":
		return &p.Filters.Name, nil
	case "category":
		return &p.Filters.Category, nil
	case "environment":
		return &p.Filters.Environment, nil
	case "perimeter":
		return &p.Filters.Perimeter, nil
	case "job":
		return &p.Filters.Job, nil
	case "tags":
		return &p.Filters.Tags, nil
	case "ids":
		return &p.Filters.IDs, nil
	case "roots":
		return &p.Filters.Roots, nil
	case "latest":
		return &p.Filters.Latest, nil

	case "minutes":
		return &p.Periods.Minute.Keep, nil
	case "hours":
		return &p.Periods.Hour.Keep, nil
	case "days":
		return &p.Periods.Day.Keep, nil
	case "weeks":
		return &p.Periods.Week.Keep, nil
	case "months":
		return &p.Periods.Month.Keep, nil
	case "years":
		return &p.Periods.Year.Keep, nil
	case "mondays":
		return &p.Periods.Monday.Keep, nil
	case "tuesdays":
		return &p.Periods.Tuesday.Keep, nil
	case "wednesdays":
		return &p.Periods.Wednesday.Keep, nil
	case "thursdays":
		return &p.Periods.Thursday.Keep, nil
	case "fridays":
		return &p.Periods.Friday.Keep, nil
	case "saturdays":
		return &p.Periods.Saturday.Keep, nil
	case "sundays":
		return &p.Periods.Sunday.Keep, nil

	case "per-minute":
		return &p.Periods.Minute.Cap, nil
	case "per-hour":
		return &p.Periods.Hour.Cap, nil
	case "per-day":
		return &p.Periods.Day.Cap, nil
	case "per-week":
		return &p.Periods.Week.Cap, nil
	case "per-month":
		return &p.Periods.Month.Cap, nil
	case "per-year":
		return &p.Periods.Year.Cap, nil
	case "per-monday":
		return &p.Periods.Monday.Cap, nil
	case "per-tuesday":
		return &p.Periods.Tuesday.Cap, nil
	case "per-wednesday":
		return &p.Periods.Wednesday.Cap, nil
	case "per-thursday":
		return &p.Periods.Thursday.Cap, nil
	case "per-friday":
		return &p.Periods.Friday.Cap, nil
	case "per-saturday":
		return &p.Periods.Saturday.Cap, nil
	case "per-sunday":
		return &p.Periods.Sunday.Cap, nil

	default:
		return nil, fmt.Errorf("invalid key")
	}
}

func (c *policiesConfig) Set(name string, key string, value string) error {
	field, err := c.locateField(name, key)
	if err != nil {
		return err
	}

	switch p := field.(type) {
	case *int:
		return c.setInt(value, p)
	case *time.Time:
		return c.setTime(value, p)
	case *string:
		*p = value
		return nil
	case *[]string:
		return c.setStringList(value, p)
	case *bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid value: %w", err)
		}
		*p = b
		return nil
	default:
		return fmt.Errorf("invalid field type")
	}
}

func (c *policiesConfig) Unset(name string, key string) error {
	field, err := c.locateField(name, key)
	if err != nil {
		return err
	}

	switch p := field.(type) {
	case *int:
		*p = 0
	case *time.Time:
		*p = time.Time{}
	case *string:
		*p = ""
	case *[]string:
		*p = nil
	case *bool:
		*p = false
	default:
		return fmt.Errorf("invalid field type")
	}

	return nil
}

func (c *policiesConfig) Remove(name string) {
	delete(c.Policies, name)
}

func (c *policiesConfig) SaveToFile(filename string) error {
	tmpFile, err := os.CreateTemp(filepath.Dir(filename), filepath.Base(filename)+".*.tmp")
	if err != nil {
		return err
	}
	err = yaml.NewEncoder(tmpFile).Encode(c)
	tmpFile.Close()
	if err == nil {
		err = os.Rename(tmpFile.Name(), filename)
	}
	os.Remove(tmpFile.Name())
	return err
}

func (c *policiesConfig) Load(rd io.Reader) error {
	return yaml.NewDecoder(rd).Decode(c)
}

func (c *policiesConfig) Dump(w io.Writer, format string, names []string) error {

	if len(names) == 0 {
		for name := range c.Policies {
			names = append(names, name)
		}
	}

	for _, name := range names {
		if !c.Has(name) {
			return fmt.Errorf("entry %q not found", name)
		}
		var err error
		switch format {
		case "json":
			err = json.NewEncoder(w).Encode(map[string]*locate.LocateOptions{name: c.Policies[name]})
		case "yaml":
			err = yaml.NewEncoder(w).Encode(map[string]*locate.LocateOptions{name: c.Policies[name]})
		default:
			return fmt.Errorf("unknown format %q", format)
		}
		if err != nil {
			return fmt.Errorf("failed to encode entry %q: %w", name, err)
		}
	}

	return nil
}

func LoadPolicyConfigFile(filename string) (*policiesConfig, error) {
	var cfg policiesConfig
	cfg.Version = "v1.0.0"
	cfg.Policies = make(map[string]*locate.LocateOptions)

	rd, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return &cfg, nil
		}
		return nil, err
	}
	defer rd.Close()

	if err := cfg.Load(rd); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *policiesConfig) ApplyConfig(name string, po *locate.LocateOptions) {
	p, ok := c.Policies[name]
	if !ok {
		return
	}
	*po = *p
}

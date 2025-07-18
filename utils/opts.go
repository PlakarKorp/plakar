package utils

import (
	"fmt"
	"strings"
)

type OptsFlag struct {
	opts map[string]string
}

func NewOptsFlag(opts map[string]string) *OptsFlag {
	return &OptsFlag{opts}
}

func (o *OptsFlag) String() string {
	if o.opts == nil {
		return ""
	}

	var b strings.Builder
	var cont bool
	for k, v := range o.opts {
		if cont {
			b.WriteByte(' ')
		}
		cont = true
		fmt.Fprintf(&b, "%s=%s", k, v)
	}

	return b.String()
}

func (o *OptsFlag) Set(s string) error {
	k, v, found := strings.Cut(s, "=")
	if !found {
		v = "true"
	}
	o.opts[k] = v
	return nil
}

// Type is only used in help text
func (o *OptsFlag) Type() string {
	return "option=value"
}

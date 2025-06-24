package reporting

import (
	"testing"
)

func TestEmit(t *testing.T) {
	reporter := NewReporter(nil, false, nil, nil)
	reporter.TaskStart("blah", "baz")
	reporter.TaskDone()
}

package reporting

import (
	"testing"
)

func TestEmit(t *testing.T) {

	reporter := NewReporter("http://localhost:8080/report", nil)
	reporter.TaskStart("blah", "baz")
	reporter.TaskDone()
}

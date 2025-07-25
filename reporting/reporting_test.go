package reporting

import (
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
)

func TestEmit(t *testing.T) {
	ctx := appcontext.NewAppContext()
	reporter := NewReporter(ctx)
	report := reporter.NewReport()
	report.SetIgnore()
	report.TaskStart("blah", "baz")
	report.TaskDone()
	reporter.StopAndWait()
}

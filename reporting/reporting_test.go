package reporting

import (
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
)

func TestEmit(t *testing.T) {
	ctx := appcontext.NewAppContext()
	reporter := NewReporter(ctx, false)
	report := reporter.NewReport()
	report.TaskStart("blah", "baz")
	report.TaskDone()
	reporter.StopAndWait()
}

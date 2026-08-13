package cp

import (
	"os"
	"slices"
	"testing"

	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/kloset/objects"
)

func TestResolveLocation(t *testing.T) {
	nothing := func(string) (map[string]string, bool) {
		t.Helper()
		t.Error("the configuration was consulted for a plain location")
		return nil, false
	}

	for _, arg := range []string{"/var/www", "s3://bucket/path", "src"} {
		conf, err := resolve(arg, nothing)
		if err != nil {
			t.Fatalf("resolve(%q): %v", arg, err)
		}
		if conf["location"] != arg {
			t.Errorf("resolve(%q): location is %q", arg, conf["location"])
		}
	}
}

func TestResolveConfigured(t *testing.T) {
	lookup := func(name string) (map[string]string, bool) {
		if name != "mysrc" {
			return nil, false
		}
		return map[string]string{"location": "/srv/data", "foo": "bar"}, true
	}

	conf, err := resolve("@mysrc", lookup)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if conf["location"] != "/srv/data" {
		t.Errorf("location is %q", conf["location"])
	}
	// the other settings have to be handed over to the connector too.
	if conf["foo"] != "bar" {
		t.Errorf("foo is %q", conf["foo"])
	}
}

func TestResolveUnknown(t *testing.T) {
	missing := func(string) (map[string]string, bool) { return nil, false }

	if _, err := resolve("@nope", missing); err == nil {
		t.Error("resolving an unknown name succeeded")
	}

	// a configured entry without a location is not usable either.
	nolocation := func(string) (map[string]string, bool) {
		return map[string]string{"foo": "bar"}, true
	}
	if _, err := resolve("@nope", nolocation); err == nil {
		t.Error("resolving an entry without a location succeeded")
	}
}

// feedOrder runs the records through feed() and reports the order they
// were handed over in.
func feedOrder(in []*connectors.Record, root string) []*connectors.Record {
	records := make(chan *connectors.Record, len(in))
	for _, record := range in {
		records <- record
	}
	close(records)

	var got []*connectors.Record
	feed(records, root,
		func(record *connectors.Record) { got = append(got, record) },
		func(*connectors.Record) {})
	return got
}

func TestRebase(t *testing.T) {
	for _, tc := range []struct {
		pathname, root, want string
		ok                   bool
	}{
		// the root itself keeps its name, the copy lands under it.
		{"/private/etc", "/private/etc", "/etc", true},
		{"/private/etc/hosts", "/private/etc", "/etc/hosts", true},
		{"/private/etc/ssh/config", "/private/etc", "/etc/ssh/config", true},

		// the directories leading to the root are not copied.
		{"/private", "/private/etc", "", false},
		{"/", "/private/etc", "", false},
		{"/elsewhere", "/private/etc", "", false},

		// a root at the top level has nothing to strip.
		{"/srv", "/srv", "/srv", true},
		{"/srv/data", "/srv", "/srv/data", true},
	} {
		got, ok := rebase(tc.pathname, tc.root)
		if ok != tc.ok || got != tc.want {
			t.Errorf("rebase(%q, %q) = %q, %v; want %q, %v",
				tc.pathname, tc.root, got, ok, tc.want, tc.ok)
		}
	}
}

// The whole point of the reordering: whatever order the importer emits
// the tree in, a directory has to reach the exporter before anything it
// holds, and nothing may be dropped along the way.
func TestFeedSendsParentsFirst(t *testing.T) {
	dir := func(path string) *connectors.Record {
		return &connectors.Record{
			Pathname: path,
			FileInfo: objects.FileInfo{Lmode: os.ModeDir | 0755},
		}
	}
	file := func(path string) *connectors.Record {
		return &connectors.Record{
			Pathname: path,
			FileInfo: objects.FileInfo{Lmode: 0644},
		}
	}

	// deliberately hostile: every child before its parent.
	in := []*connectors.Record{
		file("/srv/a/b/deep.txt"),
		dir("/srv/a/b"),
		file("/srv/a/one.txt"),
		dir("/srv/a"),
		file("/srv/root.txt"),
		dir("/srv"),
	}

	got := feedOrder(in, "/srv")

	sent := map[string]bool{"/": true}
	for _, record := range got {
		if parent := parentOf(record.Pathname); parent != "" && !sent[parent] {
			t.Errorf("%s was sent before its parent %s", record.Pathname, parent)
		}
		sent[record.Pathname] = true
	}

	// nothing the importer emitted may go missing on the way.
	for _, record := range in {
		if !sent[record.Pathname] {
			t.Errorf("%s was never sent", record.Pathname)
		}
	}
}

// Importers serving a remote root -- sftp, s3, ... -- hand out that
// root and nothing above it, so cp has to make up the directories
// leading to it or the exporter has nowhere to put the tree.
func TestFeedMakesUpMissingParents(t *testing.T) {
	got := feedOrder([]*connectors.Record{
		{
			Pathname: "/private/etc",
			FileInfo: objects.FileInfo{Lmode: os.ModeDir | 0755},
		},
		{
			Pathname: "/private/etc/hosts",
			FileInfo: objects.FileInfo{Lmode: 0644},
		},
	}, "/private/etc")

	var paths []string
	for _, record := range got {
		paths = append(paths, record.Pathname)
	}

	// rebased under the root, with the "/etc" the importer never
	// reported made up so that hosts has somewhere to go.
	want := []string{"/etc", "/etc/hosts"}
	if !slices.Equal(paths, want) {
		t.Errorf("sent %v, want %v", paths, want)
	}
}

// The directories between the root and an entry are made up when the
// importer skips them, so that the exporter can mkdir(2) its way down.
func TestFeedMakesUpIntermediateDirectories(t *testing.T) {
	got := feedOrder([]*connectors.Record{
		{
			Pathname: "/srv/data/deep/nested/file.txt",
			FileInfo: objects.FileInfo{Lmode: 0644},
		},
	}, "/srv/data")

	var paths []string
	for _, record := range got {
		paths = append(paths, record.Pathname)
	}

	want := []string{"/data", "/data/deep", "/data/deep/nested", "/data/deep/nested/file.txt"}
	if !slices.Equal(paths, want) {
		t.Fatalf("sent %v, want %v", paths, want)
	}

	// the made up ones have to look like directories, or the exporter
	// writes them out as empty files.
	for _, record := range got[:3] {
		if !record.FileInfo.Lmode.IsDir() {
			t.Errorf("%s was not made up as a directory", record.Pathname)
		}
	}
}

// A directory the importer does report keeps its own metadata rather
// than the one cp would have made up for it.
func TestFeedPrefersReportedDirectories(t *testing.T) {
	reported := &connectors.Record{
		Pathname: "/srv/data",
		FileInfo: objects.FileInfo{Lmode: os.ModeDir | 0700},
	}

	got := feedOrder([]*connectors.Record{reported}, "/srv/data")
	if len(got) != 1 {
		t.Fatalf("sent %d records, want 1", len(got))
	}
	if got[0].FileInfo.Lmode != reported.FileInfo.Lmode {
		t.Errorf("mode is %v, want the reported %v",
			got[0].FileInfo.Lmode, reported.FileInfo.Lmode)
	}
}

func TestParentOf(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"/", ""},
		{"/tmp", "/"},
		{"/tmp/foo", "/tmp"},
		{"/a/b/c", "/a/b"},
		{"relative/path", ""},
	} {
		if got := parentOf(tc.in); got != tc.want {
			t.Errorf("parentOf(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

package tui

import (
	"fmt"
	"time"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
)

var applications = map[string]func(*appcontext.AppContext, *Application, *repository.Repository) tea.Model{
	"import": newGenericModel,
	"export": newGenericModel,
}

type Application struct {
	ctx   *appcontext.AppContext
	job   uuid.UUID
	name  string
	state *State
	done  chan struct{} // closed when Bubbletea program exits
	prog  *tea.Program
	err   error

	debounceStat time.Time
	lastStat     string
}

type State struct {
	startTime  time.Time
	snapshotID string

	timerBegin time.Time
	timerDone  bool

	phase  string
	detail string

	gotSummary       bool
	summaryPath      uint64
	summaryFile      uint64
	summaryDirectory uint64
	summarySymlink   uint64
	summaryXattr     uint64
	summarySize      uint64

	// counts (event-driven, no per-path memory)
	countPath       uint64
	countPathOk     uint64
	countPathError  uint64
	countPathCached uint64

	countDir       uint64
	countDirOk     uint64
	countDirError  uint64
	countDirCached uint64

	countFile       uint64
	countFileOk     uint64
	countFileError  uint64
	countFileCached uint64

	countXattr       uint64
	countXattrOk     uint64
	countXattrError  uint64
	countXattrCached uint64

	countSymlink       uint64
	countSymlinkOk     uint64
	countSymlinkError  uint64
	countSymlinkCached uint64

	countFileSize      int64
	countDirectorySize int64
	countXattrSize     int64
	countCachedSize    uint64

	lastItem string
	errors   []string
	logs     []string
}

func newApplicationState() *State {
	return &State{
		lastItem: "",
		errors:   []string{},
		logs:     []string{},
	}
}

func newApplication(ctx *appcontext.AppContext, name string, repo *repository.Repository) *Application {
	done := make(chan struct{})

	modelFunc, ok := applications[name]
	if !ok {
		return nil
	}

	capp := &Application{
		ctx:   ctx,
		name:  name,
		done:  done,
		state: newApplicationState(),
	}
	capp.prog = tea.NewProgram(modelFunc(ctx, capp, repo))

	go func() {
		defer close(done)
		_, err := capp.prog.Run()
		if err != nil {
			capp.err = err
		}
	}()

	return capp
}

func (app *Application) Stop() {
	if app == nil {
		return
	}

	if app.prog != nil {
		app.prog.Quit()
	}

	<-app.done
}

func (s *State) Update(e Event) {
	switch e.Type {
	case "workflow.start":
		s.startTime = time.Now()
		s.snapshotID = fmt.Sprintf("%x", e.Snapshot[0:4])

	case "workflow.end":

	case "path":
		if p, ok := e.Data["path"].(string); ok {
			s.lastItem = p
		}
		s.countPath++

	case "path.ok":
		s.countPathOk++

	case "path.error":
		if p, ok := e.Data["path"].(string); ok {
			s.errors = append(s.errors, fmt.Sprintf("%s %s: %s", crossMark, p, e.Data["error"]))
		}
		s.countPathError++

	case "path.cached":
		s.countPathCached++

	case "directory":
		s.countDir++

	case "directory.ok":
		s.countDirOk++

	case "directory.error":
		s.countDirError++

	case "directory.cached":
		s.countDirCached++

	case "file":
		s.countFile++

	case "file.ok":
		s.countFileOk++
		fileinfo := e.Data["fileinfo"].(objects.FileInfo)
		s.countFileSize += fileinfo.Size()

	case "file.error":
		s.countFileError++

	case "file.cached":
		s.countFileCached++
		fileinfo := e.Data["fileinfo"].(objects.FileInfo)
		s.countCachedSize += uint64(fileinfo.Size())

	case "xattr":
		s.countXattr++

	case "xattr.ok":
		s.countXattrOk++
		size := e.Data["size"].(int64)
		s.countXattrSize += size

	case "xattr.error":
		s.countXattrError++

	case "xattr.cached":
		s.countXattrCached++
		size := e.Data["size"].(int64)
		s.countCachedSize += uint64(size)

	case "symlink":
		s.countSymlink++

	case "symlink.ok":
		s.countSymlinkOk++

	case "symlink.error":
		s.countSymlinkError++

	case "symlink.cached":
		s.countSymlinkCached++

	case "fs.summary":
		s.gotSummary = true
		s.summaryFile = e.Data["files"].(uint64)
		s.summaryDirectory = e.Data["directories"].(uint64)
		s.summarySymlink = e.Data["symlinks"].(uint64)
		s.summaryXattr = e.Data["xattrs"].(uint64)
		s.summarySize = e.Data["size"].(uint64)
		s.summaryPath = s.summaryFile + s.summaryDirectory + s.summarySymlink + s.summaryXattr

	case "snapshot.import.start":
		s.phase = "processing"
		s.timerBegin = time.Now()

	case "snapshot.import.done":
		s.detail = ""
		s.timerDone = true

	case "snapshot.vfs.start":
		s.lastItem = ""
		s.phase = "building VFS"

	case "snapshot.vfs.end":
		s.phase = ""
		s.detail = ""
		s.lastItem = ""

	case "snapshot.index.start":
		s.lastItem = ""
		s.phase = "indexing"

	case "snapshot.index.end":
		s.lastItem = ""
		s.phase = ""
		s.detail = ""

	case "snapshot.commit.start":
		s.lastItem = ""
		s.phase = "committing"

	case "result":
		s.lastItem = ""
		s.phase = "completed"
		s.detail = fmt.Sprintf(
			"size=%s errors=%d duration=%s",
			humanize.IBytes(e.Data["size"].(uint64)),
			e.Data["errors"].(uint64),
			e.Data["duration"].(time.Duration),
		)
	}
}

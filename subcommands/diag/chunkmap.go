package diag

import (
	"bufio"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
)

type DiagChunkmap struct {
	subcommands.SubcommandBase

	SnapshotPaths []string
	HTMLOutput    string
}

func (cmd *DiagChunkmap) CobraCommand() *cobra.Command {
	c := &cobra.Command{
		Use: "diag chunkmap",
	}
	c.Flags().StringVar(&cmd.HTMLOutput, "html", "", "write HTML visualization to file")
	return c
}

func (cmd *DiagChunkmap) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	if len(rest) < 2 {
		return fmt.Errorf("usage: %s chunkmap [--html FILE] SNAPSHOT:PATH SNAPSHOT:PATH [...]", "diag chunkmap")
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.SnapshotPaths = rest

	return nil
}

type chunkmapFile struct {
	label  string
	chunks []objects.Chunk
}

func (cmd *DiagChunkmap) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	chunksCount := make(map[objects.MAC]int)
	files := make([]chunkmapFile, 0, len(cmd.SnapshotPaths))

	for _, snapshotPath := range cmd.SnapshotPaths {
		err := func() error {
			snap, pathname, err := locate.OpenSnapshotByPath(repo, snapshotPath)
			if err != nil {
				return fmt.Errorf("failed to open snapshot %s: %w", snapshotPath, err)
			}
			defer snap.Close()

			fs, err := snap.Filesystem()
			if err != nil {
				return fmt.Errorf("failed to get filesystem for snapshot %s: %w", snapshotPath, err)
			}

			entry, err := fs.GetEntry(pathname)
			if err != nil {
				return fmt.Errorf("failed to get entry %s in snapshot %s: %w", pathname, snapshotPath, err)
			}

			if entry.ResolvedObject == nil {
				return fmt.Errorf("no object found for entry %s in snapshot %s", pathname, snapshotPath)
			}

			seen := make(map[objects.MAC]struct{})
			for _, chunk := range entry.ResolvedObject.Chunks {
				if _, ok := seen[chunk.ContentMAC]; !ok {
					seen[chunk.ContentMAC] = struct{}{}
					chunksCount[chunk.ContentMAC]++
				}
			}

			files = append(files, chunkmapFile{
				label:  snapshotPath,
				chunks: entry.ResolvedObject.Chunks,
			})
			return nil
		}()
		if err != nil {
			return 1, err
		}
	}

	if cmd.HTMLOutput != "" {
		return writeChunkmapHTML(cmd.HTMLOutput, files, chunksCount)
	}

	return writeChunkmapText(ctx, files, chunksCount)
}

// shareRatio returns 0.0 (unique) to 1.0 (present in all files).
// shareCount includes the file itself; total is the number of files provided.
func shareRatio(shareCount, total int) float64 {
	if total <= 1 {
		return 1.0
	}
	return float64(shareCount-1) / float64(total-1)
}

// ansiColorForRatio returns an ANSI 256-color escape for a red→green gradient.
func ansiColorForRatio(ratio float64) string {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	r := int(5 * (1.0 - ratio))
	g := int(5 * ratio)
	return fmt.Sprintf("\033[38;5;%dm", 16+36*r+6*g)
}

// htmlColorForRatio returns a CSS hsl() color for a red→green gradient.
func htmlColorForRatio(ratio float64) template.CSS {
	return template.CSS(fmt.Sprintf("hsl(%.1f,70%%,45%%)", ratio*120))
}

const (
	ansiReset = "\033[0m"
	blockChar = "█"
)

func writeChunkmapText(ctx *appcontext.AppContext, files []chunkmapFile, chunksCount map[objects.MAC]int) (int, error) {
	total := len(files)
	for _, f := range files {
		fullyShared, partiallyShared, unique := 0, 0, 0
		var sb strings.Builder

		for _, chunk := range f.chunks {
			sc := chunksCount[chunk.ContentMAC]

			switch sc {
			case total:
				fullyShared++
			case 1:
				unique++
			default:
				partiallyShared++
			}
			sb.WriteString(ansiColorForRatio(shareRatio(sc, total)))
			sb.WriteString(blockChar)
		}
		sb.WriteString(ansiReset)

		fmt.Fprintf(ctx.Stdout, "%s: %d chunks, %d in all (%d partial, %d unique)\n",
			f.label, len(f.chunks), fullyShared, partiallyShared, unique)
		fmt.Fprintln(ctx.Stdout, sb.String())
		fmt.Fprintln(ctx.Stdout)
	}
	return 0, nil
}

type chunkmapChunkData struct {
	Index      int
	ShareCount int
	Total      int
	MAC        objects.MAC
}

type chunkmapStats struct {
	NChunks         int
	FullyShared     int
	PartiallyShared int
	Unique          int
}

type chunkmapFileData struct {
	Label  string
	Stats  chunkmapStats
	Chunks []chunkmapChunkData
}

type chunkmapTemplateData struct {
	Files []chunkmapFileData
}

var chunkmapTemplate = template.Must(template.New("chunkmap").Funcs(template.FuncMap{
	"hslColor": func(shareCount, total int) template.CSS {
		return htmlColorForRatio(shareRatio(shareCount, total))
	},
	"macHex": func(mac objects.MAC) string {
		return fmt.Sprintf("%x", mac)
	},
}).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Plakar Chunk Map</title>
<style>
body { font-family: monospace; background: #1e1e1e; color: #ccc; padding: 20px; }
h1 { color: #fff; }
.legend { margin-bottom: 32px; }
.legend-label { font-size: 13px; color: #aaa; margin-bottom: 6px; }
.gradient-bar { height: 14px; width: 300px; border-radius: 3px;
  background: linear-gradient(to right, hsl(0,70%,45%), hsl(60,70%,45%), hsl(120,70%,45%)); }
.gradient-ends { display: flex; justify-content: space-between; width: 300px;
  font-size: 12px; color: #888; margin-top: 4px; }
.file { margin: 24px 0; }
.file-label { font-size: 14px; color: #aaa; margin-bottom: 6px; }
.summary { font-size: 13px; color: #888; margin-bottom: 8px; }
.chunks { display: flex; flex-wrap: wrap; gap: 2px; }
.chunk { width: 12px; height: 12px; border-radius: 2px; cursor: default; }
</style>
</head>
<body>
<h1>Chunk Map</h1>
<div class="legend">
  <div class="legend-label">Sharing ratio (red = unique to this file, green = present in all files)</div>
  <div class="gradient-bar"></div>
  <div class="gradient-ends"><span>unique</span><span>all files</span></div>
</div>
{{range .Files -}}
<div class="file">
  <div class="file-label">{{.Label}}</div>
  <div class="summary">{{.Stats.NChunks}} chunks &mdash; {{.Stats.FullyShared}} in all files, {{.Stats.PartiallyShared}} partial, {{.Stats.Unique}} unique</div>
  <div class="chunks">
    {{range .Chunks -}}
    <div class="chunk" style="background:{{hslColor .ShareCount .Total}}" title="chunk {{.Index}}: {{.ShareCount}}/{{.Total}} files, {{macHex .MAC}}"></div>
    {{end -}}
  </div>
</div>
{{end -}}
</body>
</html>
`))

func writeChunkmapHTML(outputPath string, files []chunkmapFile, chunksCount map[objects.MAC]int) (int, error) {
	f, err := os.Create(outputPath)
	if err != nil {
		return 1, fmt.Errorf("cannot create HTML file: %w", err)
	}
	defer f.Close()

	total := len(files)
	data := chunkmapTemplateData{Files: make([]chunkmapFileData, 0, len(files))}

	for _, file := range files {
		chunks := make([]chunkmapChunkData, len(file.chunks))
		var s chunkmapStats
		s.NChunks = len(file.chunks)
		for i, chunk := range file.chunks {
			sc := chunksCount[chunk.ContentMAC]
			switch sc {
			case total:
				s.FullyShared++
			case 1:
				s.Unique++
			default:
				s.PartiallyShared++
			}
			chunks[i] = chunkmapChunkData{
				Index:      i,
				ShareCount: sc,
				Total:      total,
				MAC:        chunk.ContentMAC,
			}
		}
		data.Files = append(data.Files, chunkmapFileData{
			Label:  file.label,
			Stats:  s,
			Chunks: chunks,
		})
	}

	bw := bufio.NewWriter(f)
	if err := chunkmapTemplate.Execute(bw, data); err != nil {
		return 1, fmt.Errorf("failed to render HTML: %w", err)
	}
	if err := bw.Flush(); err != nil {
		return 1, fmt.Errorf("failed to flush HTML: %w", err)
	}
	return 0, nil
}

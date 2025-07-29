package testing

import (
	"context"
	"strings"

	"github.com/PlakarKorp/kloset/snapshot/importer"
)

type MockImporter struct {
	location string
	files    map[string]MockFile
	gen      func(chan<- *importer.ScanResult)
}

func init() {
	importer.Register("mock", 0, NewMockImporter)
}

func NewMockImporter(appCtx context.Context, opts *importer.Options, name string, config map[string]string) (importer.Importer, error) {
	return &MockImporter{
		location: config["location"],
	}, nil

}

func (p *MockImporter) SetFiles(files []MockFile) {
	p.files = make(map[string]MockFile)
	for _, file := range files {
		if !strings.HasPrefix(file.Path, "/") {
			file.Path = "/" + file.Path
		}

		// create all the leading directories
		parts := strings.Split(file.Path, "/")
		for i := range parts {
			comp := strings.Join(parts[:i], "/")
			if comp == "" {
				comp = "/"
			}
			if _, ok := p.files[comp]; !ok {
				p.files[comp] = NewMockDir(comp)
			}
		}

		p.files[file.Path] = file
	}
}

func (p *MockImporter) SetGenerator(gen func(chan<- *importer.ScanResult)) {
	p.gen = gen
}

func (p *MockImporter) Origin(ctx context.Context) (string, error) {
	return "mock", nil
}

func (p *MockImporter) Type(ctx context.Context) (string, error) {
	return "mock", nil
}

func (p *MockImporter) Scan(ctx context.Context) (<-chan *importer.ScanResult, error) {
	ch := make(chan *importer.ScanResult)
	if p.gen != nil {
		go p.gen(ch)
	} else {
		go func() {
			for _, file := range p.files {
				ch <- file.ScanResult()
			}
			close(ch)
		}()
	}
	return ch, nil
}

func (p *MockImporter) Close(ctx context.Context) error {
	return nil
}

func (p *MockImporter) Root(ctx context.Context) (string, error) {
	return "/", nil
}

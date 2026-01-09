package testing

import (
	"context"
	"strings"

	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/kloset/connectors/importer"
)

type MockImporter struct {
	location string
	files    map[string]MockFile
	gen      func(chan<- *connectors.Record)
}

func init() {
	importer.Register("mock", 0, NewMockImporter)
}

func NewMockImporter(appCtx context.Context, opts *connectors.Options, name string, config map[string]string) (importer.Importer, error) {
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

func (p *MockImporter) SetGenerator(gen func(chan<- *connectors.Record)) {
	p.gen = gen
}

func (p *MockImporter) Origin() string {
	return "mock"
}

func (p *MockImporter) Type() string {
	return "mock"
}

func (p *MockImporter) Import(ctx context.Context, records chan<- *connectors.Record, results <-chan *connectors.Result) error {
	if p.gen != nil {
		go p.gen(records)
	} else {
		go func() {
			for _, file := range p.files {
				records <- file.ScanResult()
			}
			close(records)
		}()
	}
	return nil
}

func (p *MockImporter) Ping(ctx context.Context) error {
	return nil
}

func (p *MockImporter) Close(ctx context.Context) error {
	return nil
}

func (p *MockImporter) Root() string {
	return "/"
}

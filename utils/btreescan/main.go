package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"

	"github.com/PlakarKorp/kloset/btree"
	"github.com/PlakarKorp/kloset/snapshot/importer"
	"github.com/PlakarKorp/kloset/snapshot/vfs"
	"github.com/PlakarKorp/plakar/appcontext"
	_ "github.com/PlakarKorp/plakar/connectors/fs/importer"
	_ "github.com/PlakarKorp/plakar/connectors/ftp/importer"
	_ "github.com/PlakarKorp/plakar/connectors/s3/importer"
	_ "github.com/PlakarKorp/plakar/connectors/sftp/importer"
	_ "github.com/PlakarKorp/plakar/connectors/stdio/importer"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/cockroachdb/pebble/v2"
	"github.com/vmihailenco/msgpack/v5"
)

type empty struct{}

type Node = btree.Node[string, int, empty]

type pebbleStore struct {
	counter int
	db      *pebble.DB
}

var (
	verify  bool
	xattr   bool
	dbpath  string
	order   int
	dot     string
	memprof string
	cpuprof string
)

func (l *pebbleStore) Get(i int) (*Node, error) {
	key := fmt.Sprintf("%d", i)
	bytes, closer, err := l.db.Get([]byte(key))
	if err != nil {
		return nil, err
	}
	node := &Node{}
	err = msgpack.Unmarshal(bytes, node)
	closer.Close()
	return node, err
}

func (l *pebbleStore) Update(i int, node *Node) error {
	key := fmt.Sprintf("%d", i)
	bytes, err := msgpack.Marshal(node)
	if err != nil {
		return err
	}
	return l.db.Set([]byte(key), bytes, nil)
}

func (l *pebbleStore) Put(node *Node) (int, error) {
	n := l.counter
	key := fmt.Sprintf("%d", n)
	l.counter++

	bytes, err := msgpack.Marshal(node)
	if err != nil {
		return 0, err
	}
	return n, l.db.Set([]byte(key), bytes, nil)
}

func main() {
	flag.BoolVar(&verify, "verify", false, `Whether to verify the tree at the end`)
	flag.BoolVar(&xattr, "xattr", false, `get xattr for all the files as well`)
	flag.StringVar(&dbpath, "dbpath", "/tmp/pebble", `Path to the pebble db directory; use "memory" for an in-memory btree`)
	flag.IntVar(&order, "order", 50, `Order of the btree`)
	flag.StringVar(&dot, "dot", "", `where to put the generated dot; empty for none`)
	flag.StringVar(&cpuprof, "profile-cpu", "", "profile CPU usage")
	flag.StringVar(&memprof, "profile-mem", "", "profile MEM usage")
	flag.Parse()

	if flag.NArg() != 1 {
		log.Fatal("Missing directory to import")
	}

	location := flag.Arg(0)

	if cpuprof != "" {
		f, err := os.Create(cpuprof)
		if err != nil {
			log.Fatalf("%s: could not create CPU profile: %s\n", flag.CommandLine.Name(), err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("%s: could not start CPU profile: %s\n", flag.CommandLine.Name(), err)
		}
		defer pprof.StopCPUProfile()
	}

	if memprof != "" {
		f, err := os.Create(memprof)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		runtime.GC()    // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatalf("%s: could not write MEM profile: %d\n", flag.CommandLine.Name(), err)
		}
	}

	cdir, err := utils.GetConfigDir("plakar")
	if err != nil {
		log.Fatal("failed to get plakar config dir: ", err)
	}

	config, err := utils.LoadOldConfigIfExists(filepath.Join(cdir, "plakar.yml"))
	if err != nil {
		log.Fatal("could not load config file: ", err)
	}

	var importerSource map[string]string
	if strings.HasPrefix(location, "@") {
		var ok bool
		importerSource, ok = config.GetSource(location[1:])
		if !ok {
			log.Fatal("could not load source: ", location[1:])
		}
	} else {
		importerSource = map[string]string{"location": location}
	}

	var store btree.Storer[string, int, empty]
	if dbpath == "memory" {
		store = &btree.InMemoryStore[string, empty]{}
	} else {
		os.Remove(dbpath)
		if err := os.MkdirAll(dbpath, 0755); err != nil {
			log.Fatalf("can't mkdirall %s: %s", dbpath, err)
		}
		db, err := pebble.Open(dbpath, nil)
		if err != nil {
			log.Fatal("failed to open the pebble:", err)
		}
		store = &pebbleStore{db: db}
		defer os.Remove(dbpath)
	}

	idx, err := btree.New(store, vfs.PathCmp, order)
	if err != nil {
		log.Fatal("Failed to create the btree:", err)
	}

	ctx := appcontext.NewAppContext()

	imp, err := importer.NewImporter(ctx.GetInner(), ctx.ImporterOpts(), importerSource)
	if err != nil {
		log.Fatal("new fs importer failed:", err)
	}

	if err := doScan(imp, idx); err != nil {
		log.Fatal(err)
	}
}

func doScan(imp importer.Importer, idx *btree.BTree[string, int, empty]) error {
	scan, err := imp.Scan(context.Background())
	if err != nil {
		return fmt.Errorf("fs scan failed: %s", err)
	}

	var items uint64
	log.Println("starting the scan")
	for record := range scan {
		switch {
		case record.Error != nil:
			log.Print("failed to scan:", record.Error.Pathname)
			continue
		case record.Record != nil:
			path := record.Record.Pathname
			if err := idx.Insert(path, empty{}); err != nil && err != btree.ErrExists {
				record.Record.Reader.Close()
				return fmt.Errorf("failed to insert %s: %v", path, err)
			}
			items++

			if xattr && record.Record.IsXattr {
				log.Println(path, "found xattr named", record.Record.XattrName)
			}

			record.Record.Reader.Close()
		default:
			return fmt.Errorf("got unknown scanrecord %v", record)
		}
	}
	log.Println("scan finished.", items, "items scanned")

	if dot != "" {
		fp, err := os.Create(dot)
		if err != nil {
			log.Printf("failed to open %s: %v", dot, err)
		}
		fmt.Fprintln(fp, "digraph g {")
		idx.Dot(fp, false)
		fmt.Fprintln(fp, "}")
	}

	if verify {
		if err := idx.Verify(); err != nil {
			return fmt.Errorf("verify failed: %s", err)
		}
	}

	return nil
}

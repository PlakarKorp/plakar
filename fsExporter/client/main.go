package main

import (
	"context"
	"fmt"
	"time"

	"github.com/PlakarKorp/go-kloset-sdk/pkg/exporter"
	"github.com/PlakarKorp/plakar/objects"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type fakeFile struct {
	name string
	content []byte
	info objects.FileInfo
}

var fakeFiles = []fakeFile{
	{
		name:    "/tmp/file1.txt",
		content: []byte("This is the content of file1.txt"),
		info: objects.FileInfo{
			Lname:      "file1.txt",
			Lsize:      int64(len([]byte("This is the content of file1.txt"))),
			Lmode:      0644,
			LmodTime:   time.Now(),
			Ldev:       1,
			Lino:       1,
			Luid:       1000,
			Lgid:       1000,
			Lnlink:     1,
			Lusername:  "user",
			Lgroupname: "group",
			Flags:      0,
		},
	},
	{
		name:    "/tmp/file2.log",
		content: []byte("Log entries for testing purposes"),
		info: objects.FileInfo{
			Lname:      "file2.log",
			Lsize:      int64(len([]byte("Log entries for testing purposes"))),
			Lmode:      0644,
			LmodTime:   time.Now(),
			Ldev:       1,
			Lino:       2,
			Luid:       1000,
			Lgid:       1000,
			Lnlink:     1,
			Lusername:  "user",
			Lgroupname: "group",
			Flags:      0,
		},
	},
	{
		name:    "/tmp/subdir/file3.dat",
		content: []byte{0x01, 0x02, 0x03, 0x04},
		info: objects.FileInfo{
			Lname:      "file3.dat",
			Lsize:      4,
			Lmode:      0644,
			LmodTime:   time.Now(),
			Ldev:       1,
			Lino:       3,
			Luid:       1000,
			Lgid:       1000,
			Lnlink:     1,
			Lusername:  "user",
			Lgroupname: "group",
			Flags:      0,
		},
	},
}

func main() {
	serverAddr := "localhost:50052"
	conn, err := grpc.NewClient(serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client := exporter.NewExporterClient(conn)

	ctx := context.Background()

	for _, file := range fakeFiles {
		// Create directory if it doesn't exist
		dir := file.name[:len(file.name)-len(file.info.Lname)]
		_, err := client.CreateDirectory(ctx, &exporter.CreateDirectoryRequest{Pathname: dir})
		if err != nil {
			fmt.Printf("Failed to create directory %s: %v\n", dir, err)
			continue
		}
		stream, err := client.StoreFile(ctx)
		if err != nil {
			fmt.Printf("Failed to create stream for %s: %v\n", file.name, err)
			continue
		}

		// Send file data
		req := &exporter.StoreFileRequest{
			Pathname: file.name,
			Size:     uint64(file.info.Lsize),
			Fp: &exporter.IoReader{
				Chunk: file.content,
			},
		}

		if err := stream.Send(req); err != nil {
			fmt.Printf("Failed to send file %s: %v\n", file.name, err)
			continue
		}

		// Close the stream and get response
		_, err = stream.CloseAndRecv()
		if err != nil {
			fmt.Printf("Failed to close stream for %s: %v\n", file.name, err)
			continue
		}

		fmt.Printf("Successfully stored %s\n", file.name)
	}
}

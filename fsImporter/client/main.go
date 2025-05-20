package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/PlakarKorp/go-kloset-sdk/pkg/importer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func ScanFS(client importer.ImporterClient) {
	scanStream, err := client.Scan(context.Background(), &importer.ScanRequest{})
	if err != nil {
		panic(err)
	}
	for {
		resp, err := scanStream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			panic(err)
		}
		if scanError := resp.GetError(); scanError != nil {
			fmt.Printf("[ERROR] %s: %s\n", resp.Pathname, scanError.GetMessage())
		} else if record := resp.GetRecord(); record != nil {
			fmt.Printf("[OK] %s\n", resp.Pathname)
		} else {
			panic("?? unexpected response")
		}
	}
}

func GetFileContent(client importer.ImporterClient, filename string) {
	data, err := client.Read(context.Background(), &importer.ReadRequest{Pathname: filename})
	if err != nil {
		panic(err)
	}
	for {
		resp, err := data.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			panic(err)
		}
		fmt.Printf("%s", resp.Data)
	}
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

	client := importer.NewImporterClient(conn)

	info, err := client.Info(context.Background(), &importer.InfoRequest{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Importer type: %v\n", info.Type)
	fmt.Printf("Importer origin: %v\n", info.Origin)
	fmt.Printf("Importer root: %v\n", info.Root)

	stream, err := client.Scan(context.Background(), &importer.ScanRequest{})
	if err != nil {
		fmt.Printf("Scan error: %v\n", err)
		return
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			fmt.Println("Scan completed.")
			break
		}
		if err != nil {
			fmt.Printf("Error receiving from stream: %v\n", err)
			break
		}
		fmt.Printf("Received path: %s\n", resp.Pathname)
		content, err := client.Read(context.Background(), &importer.ReadRequest{Pathname: resp.Pathname})
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			continue
		}
		for {
			data, err := content.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Printf("Error receiving file content: %v\n", err)
				break
			}
			fmt.Printf("File content: %s\n", data.Data)
		}
	}
	// GetFileContent(client, "/Users/niluje/dev/plakar/plakar-ui/README.md")
}

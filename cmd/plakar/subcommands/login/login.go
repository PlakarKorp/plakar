/*
 * Copyright (c) 2021 Gilles Chehade <gilles@poolp.org>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package login

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/repository"
)

func init() {
	subcommands.Register("login", parse_cmd_login)
}

func parse_cmd_login(ctx *appcontext.AppContext, args []string) (subcommands.Subcommand, error) {
	var opt_nospawn bool

	flags := flag.NewFlagSet("login", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] PROVIDER\n", flags.Name())
		fmt.Fprintln(flags.Output(), "\nSupported providers:")
		fmt.Fprintln(flags.Output(), "- github")
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.BoolVar(&opt_nospawn, "no-spawn", false, "don't spawn browser")
	flags.Parse(args)

	if flags.NArg() != 1 {
		flags.Usage()
		return nil, fmt.Errorf("invalid arguments")
	}

	provider := flags.Arg(0)
	if provider != "github" {
		flags.Usage()
		return nil, fmt.Errorf("invalid provider %s", provider)
	}

	return &Login{
		Provider: provider,
		NoSpawn:  opt_nospawn,
	}, nil
}

type Login struct {
	Provider string
	NoSpawn  bool
}

func (cmd *Login) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	flow, err := NewLoginFlow()
	if err != nil {
		return 1, err
	}

	defer flow.Close()

	token, err := flow.Run(!cmd.NoSpawn)
	if err != nil {
		return 1, err
	}

	// XXX: store token in the configuration
	fmt.Printf("Authentication token: %s\n", token)

	return 0, nil
}

type loginFlow struct {
	server  *http.Server
	port    int
	channel chan string
}

// NewLoginFlow spawns a local HTTP server in a goroutine, listening for the
// OAuth callback on a random port.
func NewLoginFlow() (*loginFlow, error) {
	flow := &loginFlow{}

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, fmt.Errorf("failed to listen on a random port: %w", err)
	}
	flow.port = listener.Addr().(*net.TCPAddr).Port

	flow.channel = make(chan string)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")

		if token != "" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(loginOkHTML))
			flow.channel <- token
		} else {
			fmt.Fprintf(w, "Waiting for OAuth callback with token...")
		}
	})

	flow.server = &http.Server{Handler: mux}
	go func() {
		if err := flow.server.Serve(listener); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()
	return flow, nil
}

//go:embed loginOk.html
var loginOkHTML string

// Run sends a POST request to the login endpoint, and waits for the user to
// complete the OAuth flow.
func (flow *loginFlow) Run(spawnBrowser bool) (string, error) {
	reqBody := map[string]string{
		"provider": "github",
		"mode":     "headers",
		"redirect": fmt.Sprintf("http://localhost:%d", flow.port),
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %v", err)
	}

	// XXX: make backend URL configurable
	resp, err := http.Post("http://localhost:8080/v1/auth/login", "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("unable to get the login URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var respData struct {
		URL string `json:"URL"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", fmt.Errorf("failed to decode response JSON: %v", err)
	}

	if spawnBrowser {
		switch runtime.GOOS {
		case "windows":
			err = exec.Command("rundll32", "url.dll,FileProtocolHandler", respData.URL).Start()
		case "darwin":
			err = exec.Command("open", respData.URL).Start()
		default: // "linux", "freebsd", "openbsd", "netbsd"
			err = exec.Command("xdg-open", respData.URL).Start()
		}
	}
	if !spawnBrowser || err != nil {
		fmt.Printf("Open the following URL in your browser to login:\n\n%s\n", respData.URL)
	}

	// Wait for the token to be received
	fmt.Printf("\nWaiting for your browser to complete the login...\n")
	token := <-flow.channel

	return token, nil
}

func (flow *loginFlow) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := flow.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("error shutting down server: %v", err)
	}
	return nil
}

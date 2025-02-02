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

package rm

import (
	"flag"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/rpc"
	"github.com/PlakarKorp/plakar/rpc/rm"
)

func init() {
	subcommands.Register("rm", parse_cmd_rm)
}

func parse_cmd_rm(ctx *appcontext.AppContext, repo *repository.Repository, args []string) (rpc.RPC, error) {
	var opt_older string
	var opt_tag string
	flags := flag.NewFlagSet("rm", flag.ExitOnError)
	flags.StringVar(&opt_tag, "tag", "", "filter by tag")
	flags.StringVar(&opt_older, "older", "", "remove snapshots older than this date")
	flags.Parse(args)

	var beforeDate time.Time
	if opt_older != "" {
		now := time.Now()

		if reg, err := regexp.Compile(`^(\d)\s?([[:alpha:]]+)$`); err != nil {
			log.Fatalf("invalid regexp: %s", opt_older)
		} else {

			matches := reg.FindStringSubmatch(opt_older)
			if len(matches) != 3 {
				layouts := []string{
					time.RFC3339,
					"2006-01-02 15:04:05",
					"02 Jan 06 15:04 MST",
					"January 2, 2006 at 3:04pm (MST)",
					"06/01/02 03:04 PM",
				}
				found := false
				for _, layout := range layouts {
					parsedTime, err := time.Parse(layout, opt_older)
					if err != nil {
						continue
					} else {
						beforeDate = parsedTime
						found = true
						break
					}
				}
				if !found {
					log.Fatalf("invalid date format: %s", opt_older)
				}
			} else {
				var duration time.Duration

				if num, err := strconv.ParseInt(matches[1], 0, 64); err != nil {
					log.Fatalf("invalid date format: %s", opt_older)
				} else {
					switch strings.ToLower(matches[2]) {
					case "minutes", "minute", "mins", "min", "m":
						duration = time.Minute * time.Duration(num)
					case "hours", "hour", "h":
						duration = time.Hour * time.Duration(num)
					case "days", "day", "d":
						duration = 24 * time.Hour * time.Duration(num)
					case "weeks", "week", "w":
						duration = 7 * 24 * time.Hour * time.Duration(num)
					case "months", "month":
						duration = 31 * 24 * time.Hour * time.Duration(num)
					case "years", "year":
						duration = 365 * 24 * time.Hour * time.Duration(num)
					default:
						log.Fatalf("invalid date format: %s", opt_older)
					}
				}

				beforeDate = now.Add(-duration)
			}
		}
	}

	if flags.NArg() == 0 && opt_older == "" && opt_tag == "" {
		log.Fatalf("%s: need at least one snapshot ID to rm", flag.CommandLine.Name())
	}

	return &rm.Rm{
		RepositoryLocation: repo.Location(),
		RepositorySecret:   ctx.GetSecret(),
		Tag:                opt_tag,
		BeforeDate:         beforeDate,
		Prefixes:           flags.Args(),
	}, nil
}

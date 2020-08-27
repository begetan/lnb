package main

import (
	"github.com/urfave/cli/v2"
)

var getCommand = cli.Command{
	Name:    "get",
	Aliases: []string{"g"},
	Usage:   "balance, status",
	Subcommands: []*cli.Command{
		{
			Name:     "balance",
			Aliases:  []string{"b"},
			Usage:    "Get lightning network daemon (lnd) total channels' balance.",
			Category: "get",
			Action:   getBalance,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "all",
					Usage: "list all channels",
					Value: true,
				},
				&cli.BoolFlag{
					Name:  "active",
					Usage: "only list channels which are currently active",
				},
				&cli.BoolFlag{
					Name:  "inactive",
					Usage: "only list channels which are currently inactive",
				},
				&cli.BoolFlag{
					Name:  "public",
					Usage: "only list channels which are currently public",
				},
				&cli.BoolFlag{
					Name:  "private",
					Usage: "only list channels which are currently private",
				},
			},
		},
		{
			Name:     "status",
			Aliases:  []string{"s"},
			Usage:    "Get lightning network daemon (lnd) status.",
			Category: "get",
			Action:   getStatus,
		},
	},
}

var listCommand = cli.Command{
	Name:    "list",
	Aliases: []string{"l"},
	Usage:   "channels, contracts",
	Subcommands: []*cli.Command{
		{
			Name:     "channels",
			Usage:    "List all open channels.",
			Category: "list",
			Action:   listChannels,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "active_only",
					Usage: "only list channels which are currently active",
				},
				&cli.BoolFlag{
					Name:  "inactive_only",
					Usage: "only list channels which are currently inactive",
				},
				&cli.BoolFlag{
					Name:  "public_only",
					Usage: "only list channels which are currently public",
				},
				&cli.BoolFlag{
					Name:  "private_only",
					Usage: "only list channels which are currently private",
				},
				&cli.StringFlag{
					Name: "peer",
					Usage: "(optional) only display channels for a peer " +
						"with a 66-byte hex-encoded pubkey",
				},
			},
		},
		{
			Name:      "contracts",
			Usage:     "List all forwarded Hash Time-Locked Contracts.",
			ArgsUsage: "start_time [end_time] [index_offset] [max_events]",
			Category:  "list",
			Action:    listContracts,
			Flags: []cli.Flag{
				&cli.Int64Flag{
					Name: "start_time",
					Usage: "the starting time for the query, expressed in " +
						"seconds since the unix epoch",
				},
				&cli.Int64Flag{
					Name: "end_time",
					Usage: "the end time for the query, expressed in " +
						"seconds since the unix epoch",
				},
				&cli.Int64Flag{
					Name:  "index_offset",
					Usage: "the number of events to skip",
				},
				&cli.Int64Flag{
					Name:        "max_events",
					Usage:       "the max number of events to return",
					DefaultText: "100",
				},
				&cli.StringFlag{
					Name: "channel",
					Usage: "(optional) only display contracts for a channel " +
						"with id in bbbbbb:iiii:p format",
				},
			},
		},
	},
}

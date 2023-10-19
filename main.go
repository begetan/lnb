package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"google.golang.org/grpc"
)

const (
	defaultLndDir			= "~/.lnd"
	defaultDataDir          = "data"
	defaultChainSubDir      = "chain"
	defaultTLSCertFilename  = "tls.cert"
	defaultMacaroonFilename = "admin.macaroon"
	defaultRPCPort          = "10009"
	defaultRPCHostPort      = "localhost:" + defaultRPCPort
)

var (
	// maxMsgRecvSize is the largest message our client will receive. We
	// set this to 200MiB atm.
	maxMsgRecvSize = grpc.MaxCallRecvMsgSize(1 * 1024 * 1024 * 200)
)

func main() {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "lnb"
	app.Usage = "lighting channel balancer for lnd"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "rpcserver",
			Value: defaultRPCHostPort,
			Usage: "host:port of ln daemon",
		},
		&cli.StringFlag{
			Name:  "lnddir",
			Value: defaultLndDir,
			Usage: "path to lnd's base directory",
		},
		&cli.StringFlag{
			Name:  "chain, c",
			Usage: "the chain lnd is running on e.g. bitcoin",
			Value: "bitcoin",
		},
		&cli.StringFlag{
			Name: "network, n",
			Usage: "the network lnd is running on e.g. mainnet, " +
				"testnet, etc.",
			Value: "mainnet",
		},
		&cli.BoolFlag{
			Name:  "no-macaroons",
			Usage: "disable macaroon authentication",
		},
		&cli.StringFlag{
			Name:  "macaroonpath",
			Usage: "path to macaroon file",
		},
		&cli.Int64Flag{
			Name:  "macaroontimeout",
			Value: 60,
			Usage: "anti-replay macaroon validity time in seconds",
		},
		&cli.StringFlag{
			Name:  "macaroonip",
			Usage: "if set, lock macaroon to specific IP address",
		},
	}
	app.Commands = []*cli.Command{
		&getCommand,
		&listCommand,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "[lnb] %v\n", err)
		os.Exit(1)
	}
}

package main

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/lightninglabs/lndclient"
	"github.com/lightningnetwork/lnd/lnrpc/verrpc"
	"github.com/urfave/cli/v2"
)

const (
	defaultLndDir           = "~/.lnd"
	defaultDataDir          = "data"
	defaultChainSubDir      = "chain"
	defaultTLSCertFilename  = "tls.cert"
	defaultMacaroonFilename = "admin.macaroon"
	defaultRPCPort          = "10009"
	defaultRPCHostPort      = "localhost:" + defaultRPCPort
)

var (
	// minRequiredLndVersion is the minimum required version of lnd that
	// is compatible with the current version of the lnb.
	minRequiredLndVersion = &verrpc.Version{
		AppMajor: 0,
		AppMinor: 17,
		AppPatch: 0,
	}
)

// cleanAndExpandPath expands environment variables and leading ~ in the
// passed path, cleans the result, and returns it.
// This function is taken from https://github.com/btcsuite/btcd
func cleanAndExpandPath(path string) string {
	if path == "" {
		return ""
	}

	// Expand initial ~ to OS specific home directory.
	if strings.HasPrefix(path, "~") {
		var homeDir string
		user, err := user.Current()
		if err == nil {
			homeDir = user.HomeDir
		} else {
			homeDir = os.Getenv("HOME")
		}

		path = strings.Replace(path, "~", homeDir, 1)
	}

	// NOTE: The os.ExpandEnv doesn't work with Windows-style %VARIABLE%,
	// but the variables can still be expanded via POSIX-style $VARIABLE.
	return filepath.Clean(os.ExpandEnv(path))
}

func getClient(callerCtx context.Context, ctx *cli.Context) (*lndclient.GrpcLndServices, error) {
	// We'll now fetch the lnddir so we can make a decision  on how to
	// properly read the macaroons (if needed) and also the cert. This will
	// either be the default, or will have been overwritten by the end
	// user.
	lndDir := cleanAndExpandPath(ctx.String("lnddir"))

	network := strings.ToLower(ctx.String("network"))

	// If the macaroon path as been manually provided, then we'll only
	// target the specified file.
	var macPath string
	if ctx.String("macaroonpath") != "" {
		macPath = cleanAndExpandPath(ctx.String("macaroonpath"))
	} else {
		// Otherwise, we'll go into the path:
		// lnddir/data/chain/<chain>/<network> in order to fetch the
		// macaroon that we need.
		macPath = filepath.Join(
			lndDir, defaultDataDir, defaultChainSubDir, "bitcoin",
			network, defaultMacaroonFilename,
		)
	}

	tlsCertPath := cleanAndExpandPath(ctx.String("tlscertpath"))

	// If a custom lnd directory was set, we'll also check if custom paths
	// for the TLS cert and macaroon file were set as well. If not, we'll
	// override their paths so they can be found within the custom lnd
	// directory set. This allows us to set a custom lnd directory, along
	// with custom paths to the TLS cert and macaroon file.
	if lndDir != cleanAndExpandPath(defaultLndDir) {
		tlsCertPath = filepath.Join(lndDir, defaultTLSCertFilename)
	}

	return lndclient.NewLndServices(&lndclient.LndServicesConfig{
		LndAddress:         ctx.String("rpcserver"),
		Network:            lndclient.Network(ctx.String("network")),
		CustomMacaroonPath: macPath,
		TLSPath:            tlsCertPath,
		CheckVersion:       minRequiredLndVersion,
		CallerCtx:          callerCtx,
	})
}

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
			Name:  "tlscertpath",
			Usage: "path to lnd's tls.cert",
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

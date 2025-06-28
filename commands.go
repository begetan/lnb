package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/lightninglabs/lndclient"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnwire"
	"github.com/lightningnetwork/lnd/routing/route"
	"github.com/shopspring/decimal"
	"github.com/urfave/cli/v2"
)

// SumHTLC contains all forwarding amounts and fees for all channells
type SumHTLC map[uint64]ChanHTLC

// ChanHTLC contains formarding amounts and fees for given channell
type ChanHTLC struct {
	Day struct {
		AmountSatIn  btcutil.Amount
		AmountSatOut btcutil.Amount
		FeeMsat      lnwire.MilliSatoshi
	}
	Week struct {
		AmountSatIn  btcutil.Amount
		AmountSatOut btcutil.Amount
		FeeMsat      lnwire.MilliSatoshi
	}
	Month struct {
		AmountSatIn  btcutil.Amount
		AmountSatOut btcutil.Amount
		FeeMsat      lnwire.MilliSatoshi
	}
}

// TotalBalance contains total data for channels
type TotalBalance struct {
	Capacity      btcutil.Amount
	LocalBalance  btcutil.Amount
	RemoteBalance btcutil.Amount
	AmountIn      btcutil.Amount
	AmountOut     btcutil.Amount
	CommitFee     btcutil.Amount
	Ratio         float64
	Efficiency    float64
}

// TotalChannels contains extended total data for all channels
type TotalChannels struct {
	TotalBalance
	DayAmountSatIn    btcutil.Amount
	DayAmountSatOut   btcutil.Amount
	MonthAmountSatIn  btcutil.Amount
	MonthAmountSatOut btcutil.Amount

	DayFee   decimal.Decimal
	WeekFee  decimal.Decimal
	MonthFee decimal.Decimal
}

func getStatus(ctx *cli.Context) error {
	ctxb := context.Background()
	client, err := getClient(ctxb, ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to LND: %w", err)
	}
	defer client.Close()

	resp, err := client.Client.GetInfo(ctxb)
	if err != nil {
		return fmt.Errorf("client.GetInfo failed: %w", err)
	}

	// IdentityPubkey is printed as a list of numbers. Fix this.
	// Also use under_score style as in original lnb.
	// This is a modified version of https://pkg.go.dev/github.com/lightninglabs/lndclient@v0.19.0-10#Info
	type Info struct {
		// Version is the version that lnd is running.
		Version string `json:"version"`

		// BlockHeight is the best block height that lnd has knowledge of.
		BlockHeight uint32 `json:"block_height"`

		// IdentityPubkey is our node's pubkey.
		IdentityPubkey string `json:"identity_pubkey"`

		// Alias is our node's alias.
		Alias string `json:"alias"`

		// Network is the network we are currently operating on.
		Network string `json:"network"`

		// Uris is the set of our node's advertised uris.
		Uris []string `json:"uris"`

		// SyncedToChain is true if the wallet's view is synced to the main
		// chain.
		SyncedToChain bool `json:"synced_to_chain"`

		// SyncedToGraph is true if we consider ourselves to be synced with the
		// public channel graph.
		SyncedToGraph bool `json:"synced_to_graph"`

		// BestHeaderTimeStamp is the best block timestamp known to the wallet.
		BestHeaderTimeStamp int64 `json:"best_header_timestamp"`

		// ActiveChannels is the number of active channels we have.
		ActiveChannels uint32 `json:"num_active_channels"`

		// InactiveChannels is the number of inactive channels we have.
		InactiveChannels uint32 `json:"num_inactive_channels"`

		// PendingChannels is the number of pending channels we have.
		PendingChannels uint32 `json:"num_pending_channels"`
	}

	printRespJSON(Info{
		Version:             resp.Version,
		BlockHeight:         resp.BlockHeight,
		IdentityPubkey:      hex.EncodeToString(resp.IdentityPubkey[:]),
		Alias:               resp.Alias,
		Network:             resp.Network,
		Uris:                resp.Uris,
		SyncedToChain:       resp.SyncedToChain,
		SyncedToGraph:       resp.SyncedToGraph,
		BestHeaderTimeStamp: resp.BestHeaderTimeStamp.Unix(),
		ActiveChannels:      resp.ActiveChannels,
		InactiveChannels:    resp.InactiveChannels,
		PendingChannels:     resp.PendingChannels,
	})
	return nil
}

func getBalance(ctx *cli.Context) error {
	ctxb := context.Background()
	client, err := getClient(ctxb, ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to LND: %w", err)
	}
	defer client.Close()

	var opts []lndclient.ListChannelsOption
	if ctx.Bool("inactive_only") {
		opts = append(opts, func(r *lnrpc.ListChannelsRequest) {
			r.InactiveOnly = true
		})
	}
	if ctx.Bool("private_only") {
		opts = append(opts, func(r *lnrpc.ListChannelsRequest) {
			r.PrivateOnly = true
		})
	}

	resp, err := client.Client.ListChannels(
		ctxb, ctx.Bool("active_only"), ctx.Bool("public_only"), opts...,
	)
	if err != nil {
		return fmt.Errorf("client.ListChannels failed: %w", err)
	}

	printBalance(resp)

	// printRespJSON(resp)
	return nil
}

func listChannels(ctx *cli.Context) error {
	ctxb := context.Background()
	client, err := getClient(ctxb, ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to LND: %w", err)
	}
	defer client.Close()

	var opts []lndclient.ListChannelsOption

	// If the user requested channels with a particular key,
	// parse the provided pubkey.
	peer := ctx.String("peer")
	var peerKey []byte
	if len(peer) > 0 {
		pk, err := route.NewVertexFromStr(peer)
		if err != nil {
			return fmt.Errorf("invalid --peer pubkey: %v", err)
		}

		peerKey = pk[:]

		opts = append(opts, lndclient.WithPeer(peerKey))
	}

	if ctx.Bool("inactive_only") {
		opts = append(opts, func(r *lnrpc.ListChannelsRequest) {
			r.InactiveOnly = true
		})
	}
	if ctx.Bool("private_only") {
		opts = append(opts, func(r *lnrpc.ListChannelsRequest) {
			r.PrivateOnly = true
		})
	}

	resp, err := client.Client.ListChannels(
		ctxb, ctx.Bool("active_only"), ctx.Bool("public_only"), opts...,
	)
	if err != nil {
		return fmt.Errorf("client.ListChannels failed: %w", err)
	}

	count, err := countHTLC(ctxb, ctx, client)
	if err != nil {
		return err
	}

	printChannels(resp, count)
	// printRespJSON(resp)

	return nil
}

func listContracts(ctx *cli.Context) error {
	ctxb := context.Background()
	client, err := getClient(ctxb, ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to LND: %w", err)
	}
	defer client.Close()

	var (
		startTime, endTime     time.Time
		indexOffset, maxEvents uint32
	)
	args := ctx.Args().Slice()

	chanID := ctx.String("channel")

	// If the user requested contracts with a particular channel id,
	// parse it
	var id uint64

	if chanID != "" {
		p := strings.Split(chanID, ":")

		if len(p) == 3 {
			p1, err1 := strconv.ParseUint(p[0], 10, 32)
			p2, err2 := strconv.ParseUint(p[1], 10, 32)
			p3, err3 := strconv.ParseUint(p[2], 10, 16)

			if err1 != nil || err2 != nil || err3 != nil {
				return fmt.Errorf("invalid --channel id: %s, should be in bbbbbb:iiii:p format", chanID)
			}

			c := lnwire.ShortChannelID{
				BlockHeight: uint32(p1),
				TxIndex:     uint32(p2),
				TxPosition:  uint16(p3),
			}
			id = c.ToUint64()
			// fmt.Printf("ID: %s\n", lnwire.NewShortChanIDFromInt(id))

		} else {
			return fmt.Errorf("invalid --channel id: %s, should be in bbbbbb:iiii:p format", chanID)
		}
	}

	switch {
	case ctx.IsSet("start_time"):
		startTime = time.Unix(int64(ctx.Uint64("start_time")), 0)
	case len(args) > 0:
		startTimeUint, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("unable to decode start_time %v", err)
		}
		startTime = time.Unix(int64(startTimeUint), 0)
		args = args[1:]
	default:
		now := time.Now().UTC()
		startTime = now.Add(-time.Hour * 24 * 30)
	}

	switch {
	case ctx.IsSet("end_time"):
		endTime = time.Unix(int64(ctx.Uint64("end_time")), 0)
	case len(args) > 0:
		endTimeUint, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("unable to decode end_time: %v", err)
		}
		endTime = time.Unix(int64(endTimeUint), 0)
		args = args[1:]
	}

	switch {
	case ctx.IsSet("index_offset"):
		indexOffset = uint32(ctx.Int64("index_offset"))
	case len(args) > 0:
		i, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("unable to decode index_offset: %v", err)
		}
		args = args[1:]
		indexOffset = uint32(i)
	}

	switch {
	case ctx.IsSet("max_events"):
		maxEvents = uint32(ctx.Int64("max_events"))
	case len(args) > 0:
		m, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("unable to decode max_events: %v", err)
		}
		maxEvents = uint32(m)
	}

	req := lndclient.ForwardingHistoryRequest{
		StartTime: startTime,
		EndTime:   endTime,
		Offset:    indexOffset,
		MaxEvents: maxEvents,
	}
	resp, err := client.Client.ForwardingHistory(ctxb, req)
	if err != nil {
		return err
	}

	printContracts(resp.Events, id)
	//printRespJSON(resp)

	return nil
}

func countHTLC(callerCtx context.Context, ctx *cli.Context,
	client *lndclient.GrpcLndServices) (SumHTLC, error) {

	sum := make(SumHTLC)

	now := time.Now().UTC()

	startDay := now.Add(-time.Hour * 24)
	startWeek := now.Add(-time.Hour * 24 * 7)
	startMonth := now.Add(-time.Hour * 24 * 30)

	req := lndclient.ForwardingHistoryRequest{
		StartTime: startMonth,
		EndTime:   now,
		Offset:    0,
		MaxEvents: 50000,
	}
	resp, err := client.Client.ForwardingHistory(callerCtx, req)
	if err != nil {
		return nil, err
	}

	for _, event := range resp.Events {
		t := event.Timestamp
		if event.ChannelIn > 0 {
			m := sum[event.ChannelIn]
			if t.After(startDay) {
				m.Day.AmountSatIn += event.AmountMsatIn.ToSatoshis()
				m.Day.FeeMsat += event.FeeMsat
			}
			if t.After(startWeek) {
				m.Week.AmountSatIn += event.AmountMsatIn.ToSatoshis()
				m.Week.FeeMsat += event.FeeMsat
			}
			if t.After(startMonth) {
				m.Month.AmountSatIn += event.AmountMsatIn.ToSatoshis()
				m.Month.FeeMsat += event.FeeMsat
			}
			sum[event.ChannelIn] = m
		}
		if event.ChannelOut > 0 {
			m := sum[event.ChannelOut]
			if t.After(startDay) {
				m.Day.AmountSatOut += event.AmountMsatOut.ToSatoshis()
				m.Day.FeeMsat += event.FeeMsat
			}
			if t.After(startWeek) {
				m.Week.AmountSatOut += event.AmountMsatOut.ToSatoshis()
				m.Week.FeeMsat += event.FeeMsat
			}
			if t.After(startMonth) {
				m.Month.AmountSatOut += event.AmountMsatOut.ToSatoshis()
				m.Month.FeeMsat += event.FeeMsat
			}
			sum[event.ChannelOut] = m
		}
	}
	return sum, nil
}

func printBalance(channels []lndclient.ChannelInfo) {
	title := " Capacity " +
		"|    Local " +
		"|   Remote " +
		"| CommitFee" +
		"| Ratio " +
		"| Total In Out Amount" +
		"| Efficiency"
	line := strings.Repeat("-", len(title))

	row := "%9d |%9d |%9d |%9d | %5d%% |%9d %-9d |%5d%%"

	fmt.Printf(title + "\n")
	fmt.Printf(line + "\n")

	b := TotalBalance{}

	for _, c := range channels {
		b.Capacity += c.Capacity
		b.LocalBalance += c.LocalBalance
		b.RemoteBalance += c.RemoteBalance
		b.AmountIn += c.TotalReceived
		b.AmountOut += c.TotalSent
		b.CommitFee += c.CommitFee
	}

	if b.LocalBalance > 0 {
		b.Ratio = float64(b.LocalBalance) / float64(b.LocalBalance+b.RemoteBalance) * 100
		b.Efficiency = (float64(b.AmountIn) + float64(b.AmountOut)) / float64(b.Capacity) * 100
	}

	fmt.Printf(row+"\n",
		b.Capacity,
		b.LocalBalance,
		b.RemoteBalance,
		b.CommitFee,
		int64(math.Round(b.Ratio)),
		b.AmountIn,
		b.AmountOut,
		int64(math.Round(b.Efficiency)),
	)
}

func printChannels(channels []lndclient.ChannelInfo, sum SumHTLC) {

	t := TotalChannels{}

	// Table formaters
	title := "  Num |    Channel ID | Public Key" +
		"| Capacity |    Local |   Remote | Ratio " +
		"|   Day In Out Amount" +
		"| Month In Out Amount" +
		"|Mon Fee" +
		"| Total In Out Amount" +
		"| Effcy"
	line := strings.Repeat("-", len(title))
	row := "%5d%s|%11s |%10s |%9d |%9d |%9d |%5d%% |%9d %-9d |%9d %-9d |%6s |%9d %-9d |%5d%%"

	// Print table
	fmt.Printf(title + "\n")
	fmt.Printf(line + "\n")

	sort.SliceStable(channels, func(i, j int) bool {
		return channels[i].ChannelID > channels[j].ChannelID
	})

	for i, c := range channels {
		var ratio float64
		var efficiency float64

		totalIn := c.TotalReceived
		totalOut := c.TotalSent

		if c.LocalBalance > 0 {
			ratio = float64(c.LocalBalance) / float64(c.LocalBalance+c.RemoteBalance) * 100
			efficiency = (float64(totalIn) + float64(totalOut)) / float64(c.Capacity) * 100
		}

		var active string = "-"
		if c.Active {
			active = " "
		}

		m := lnwire.NewShortChanIDFromInt(c.ChannelID)
		mark := fmt.Sprintf("%7d:%04d:%1d", m.BlockHeight, m.TxIndex, m.TxPosition)

		var monthFee string
		monthFeeSat := decimal.NewFromInt(int64(sum[c.ChannelID].Month.FeeMsat)).Div(decimal.NewFromInt(1000))

		switch {
		case monthFeeSat.IsZero():
			monthFee = "0"
		case monthFeeSat.LessThan(decimal.NewFromInt(100)):
			monthFee = monthFeeSat.StringFixed(3)
		default:
			monthFee = monthFeeSat.StringFixedBank(0)

		}

		fmt.Printf(row+"\n",
			i+1,
			active,
			mark,
			hex.EncodeToString(c.PubKeyBytes[:4]),
			c.Capacity,
			c.LocalBalance,
			c.RemoteBalance,
			int64(math.Round(ratio)),
			sum[c.ChannelID].Day.AmountSatIn,
			sum[c.ChannelID].Day.AmountSatOut,
			sum[c.ChannelID].Month.AmountSatIn,
			sum[c.ChannelID].Month.AmountSatOut,
			monthFee,
			totalIn,
			totalOut,
			int64(math.Round(efficiency)),
		)

		t.Capacity += c.Capacity
		t.LocalBalance += c.LocalBalance
		t.RemoteBalance += c.RemoteBalance
		t.AmountIn += totalIn
		t.AmountOut += totalOut
		t.CommitFee += c.CommitFee
		t.MonthFee = t.MonthFee.Add(monthFeeSat)

		t.DayAmountSatIn += sum[c.ChannelID].Day.AmountSatIn
		t.DayAmountSatOut += sum[c.ChannelID].Day.AmountSatOut
		t.MonthAmountSatIn += sum[c.ChannelID].Month.AmountSatIn
		t.MonthAmountSatOut += sum[c.ChannelID].Month.AmountSatOut

	}
	if t.LocalBalance > 0 {
		t.Ratio = float64(t.LocalBalance) / float64(t.LocalBalance+t.RemoteBalance) * 100
		t.Efficiency = (float64(t.AmountIn) + float64(t.AmountOut)) / float64(t.Capacity) * 100
	}

	// Print total row
	fmt.Printf(line + "\n")
	fmt.Printf(row+"\n",
		len(channels),
		" ",
		"              ",
		"  ",
		t.Capacity,
		t.LocalBalance,
		t.RemoteBalance,
		int64(math.Round(t.Ratio)),
		t.DayAmountSatIn,
		t.DayAmountSatOut,
		t.MonthAmountSatIn,
		t.MonthAmountSatOut,
		t.MonthFee.StringFixedBank(0),
		t.AmountIn,
		t.AmountOut,
		int64(math.Round(t.Efficiency)),
	)
	return
}

func printContracts(contracts []lndclient.ForwardingEvent, id uint64) {
	// Table formater
	title := "  Num " +
		"|            Time           " +
		"|  Timestamp " +
		"|  Channel In   " +
		"|  Channel Out  " +
		"| Amount In" +
		"|Amount Out" +
		"| Fee Msat"

	line := strings.Repeat("-", len(title))
	row := "%5d | %24s | %10d |%10s |%10s |%9d |%9d |%6d"

	fmt.Printf(title + "\n")
	fmt.Printf(line + "\n")

	sort.SliceStable(contracts, func(i, j int) bool {
		return contracts[i].Timestamp.After(contracts[j].Timestamp)
	})

	for i, c := range contracts {
		if id != 0 && id != c.ChannelIn && id != c.ChannelOut {
			continue
		}

		tm := c.Timestamp.Format(time.RFC3339)

		in := lnwire.NewShortChanIDFromInt(c.ChannelIn)
		out := lnwire.NewShortChanIDFromInt(c.ChannelOut)

		markIn := fmt.Sprintf("%7d:%04d:%1d", in.BlockHeight, in.TxIndex, in.TxPosition)
		markOut := fmt.Sprintf("%7d:%04d:%1d", out.BlockHeight, out.TxIndex, out.TxPosition)

		fmt.Printf(row+"\n",
			i+1,
			tm,
			c.Timestamp.Unix(),
			markIn,
			markOut,
			c.AmountMsatIn.ToSatoshis(),
			c.AmountMsatOut.ToSatoshis(),
			c.FeeMsat,
		)
	}
	return
}

func printRespJSON(resp interface{}) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(resp); err != nil {
		log.Fatalf("Failed to encode response in JSON: %v", err)
	}
}

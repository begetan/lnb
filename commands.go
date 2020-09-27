package main

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/begetan/lnb/client"
	"github.com/lightninglabs/protobuf-hex-display/jsonpb"
	"github.com/lightninglabs/protobuf-hex-display/proto"
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
		AmountSatIn  uint64
		AmountSatOut uint64
		FeeMsat      uint64
	}
	Week struct {
		AmountSatIn  uint64
		AmountSatOut uint64
		FeeMsat      uint64
	}
	Month struct {
		AmountSatIn  uint64
		AmountSatOut uint64
		FeeMsat      uint64
	}
}

// TotalBalance contains total data for channels
type TotalBalance struct {
	Capacity      int64
	LocalBalance  int64
	RemoteBalance int64
	AmountIn      int64
	AmountOut     int64
	CommitFee     int64
	Ratio         float64
	Efficiency    float64
}

// TotalChannels contains extended total data for all channels
type TotalChannels struct {
	TotalBalance
	DayAmountSatIn    uint64
	DayAmountSatOut   uint64
	MonthAmountSatIn  uint64
	MonthAmountSatOut uint64

	DayFee   decimal.Decimal
	WeekFee  decimal.Decimal
	MonthFee decimal.Decimal
}

func getStatus(ctx *cli.Context) error {
	ctxb := context.Background()
	client, cleanUp := client.GetClient(ctx)
	defer cleanUp()

	req := &lnrpc.GetInfoRequest{}
	resp, err := client.GetInfo(ctxb, req)
	if err != nil {
		return err
	}

	printRespJSON(resp)
	return nil
}

func getBalance(ctx *cli.Context) error {
	ctxb := context.Background()
	client, cleanUp := client.GetClient(ctx)
	defer cleanUp()

	req := &lnrpc.ListChannelsRequest{
		ActiveOnly:   ctx.Bool("active_only"),
		InactiveOnly: ctx.Bool("inactive_only"),
		PublicOnly:   ctx.Bool("public_only"),
		PrivateOnly:  ctx.Bool("private_only"),
	}

	resp, err := client.ListChannels(ctxb, req)
	if err != nil {
		return err
	}

	printBalance(resp)

	// printRespJSON(resp)
	return nil
}

func listChannels(ctx *cli.Context) error {
	ctxb := context.Background()
	client, cleanUp := client.GetClient(ctx)
	defer cleanUp()

	peer := ctx.String("peer")

	// If the user requested channels with a particular key,
	// parse the provided pubkey.
	var peerKey []byte
	if len(peer) > 0 {
		pk, err := route.NewVertexFromStr(peer)
		if err != nil {
			return fmt.Errorf("invalid --peer pubkey: %v", err)
		}

		peerKey = pk[:]
	}

	req := &lnrpc.ListChannelsRequest{
		ActiveOnly:   ctx.Bool("active_only"),
		InactiveOnly: ctx.Bool("inactive_only"),
		PublicOnly:   ctx.Bool("public_only"),
		PrivateOnly:  ctx.Bool("private_only"),
		Peer:         peerKey,
	}

	resp, err := client.ListChannels(ctxb, req)
	if err != nil {
		return err
	}

	count, err := countHTLC(ctx)
	if err != nil {
		return err
	}

	printChannels(resp, count)
	// printRespJSON(resp)

	return nil
}

func listContracts(ctx *cli.Context) error {
	ctxb := context.Background()
	client, cleanUp := client.GetClient(ctx)
	defer cleanUp()
	var (
		startTime, endTime     uint64
		indexOffset, maxEvents uint32
		err                    error
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
		startTime = ctx.Uint64("start_time")
	case len(args) > 0:
		startTime, err = strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("unable to decode start_time %v", err)
		}
		args = args[1:]
	default:
		now := time.Now().UTC()
		startTime = uint64(now.Add(-time.Hour * 24 * 30).Unix())
	}

	switch {
	case ctx.IsSet("end_time"):
		endTime = ctx.Uint64("end_time")
	case len(args) > 0:
		endTime, err = strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("unable to decode end_time: %v", err)
		}
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

	req := &lnrpc.ForwardingHistoryRequest{
		StartTime:    startTime,
		EndTime:      endTime,
		IndexOffset:  indexOffset,
		NumMaxEvents: maxEvents,
	}

	resp, err := client.ForwardingHistory(ctxb, req)
	if err != nil {
		return err
	}

	printContracts(resp, id)
	//printRespJSON(resp)

	return nil
}

func countHTLC(ctx *cli.Context) (SumHTLC, error) {

	sum := make(SumHTLC)

	ctxb := context.Background()
	client, cleanUp := client.GetClient(ctx)
	defer cleanUp()

	now := time.Now().UTC()
	end := uint64(now.Unix())

	startDay := uint64(now.Add(-time.Hour * 24).Unix())
	startWeek := uint64(now.Add(-time.Hour * 24 * 7).Unix())
	startMonth := uint64(now.Add(-time.Hour * 24 * 30).Unix())

	req := &lnrpc.ForwardingHistoryRequest{
		StartTime:    startMonth,
		EndTime:      end,
		IndexOffset:  0,
		NumMaxEvents: 50000,
	}
	resp, err := client.ForwardingHistory(ctxb, req)
	if err != nil {
		return nil, err
	}

	for _, event := range resp.ForwardingEvents {
		t := event.Timestamp
		if event.ChanIdIn > 0 {
			m := sum[event.ChanIdIn]
			if t > startDay {
				m.Day.AmountSatIn += event.AmtIn
				m.Day.FeeMsat += event.FeeMsat
			}
			if t > startWeek {
				m.Week.AmountSatIn += event.AmtIn
				m.Week.FeeMsat += event.FeeMsat
			}
			if t > startMonth {
				m.Month.AmountSatIn += event.AmtIn
				m.Month.FeeMsat += event.FeeMsat
			}
			sum[event.ChanIdIn] = m
		}
		if event.ChanIdOut > 0 {
			m := sum[event.ChanIdOut]
			if t > startDay {
				m.Day.AmountSatOut += event.AmtOut
				m.Day.FeeMsat += event.FeeMsat
			}
			if t > startWeek {
				m.Week.AmountSatOut += event.AmtOut
				m.Week.FeeMsat += event.FeeMsat
			}
			if t > startMonth {
				m.Month.AmountSatOut += event.AmtOut
				m.Month.FeeMsat += event.FeeMsat
			}
			sum[event.ChanIdOut] = m
		}
	}
	return sum, nil
}

func printBalance(channels *lnrpc.ListChannelsResponse) {
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

	for _, c := range channels.Channels {
		b.Capacity += c.Capacity
		b.LocalBalance += c.LocalBalance
		b.RemoteBalance += c.RemoteBalance
		b.AmountIn += c.TotalSatoshisReceived
		b.AmountOut += c.TotalSatoshisSent
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

func printChannels(channels *lnrpc.ListChannelsResponse, sum SumHTLC) {

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

	sort.SliceStable(channels.Channels, func(i, j int) bool {
		return channels.Channels[i].ChanId > channels.Channels[j].ChanId
	})

	for i, c := range channels.Channels {
		var ratio float64
		var efficiency float64

		totalIn := c.TotalSatoshisReceived
		totalOut := c.TotalSatoshisSent

		if c.LocalBalance > 0 {
			ratio = float64(c.LocalBalance) / float64(c.LocalBalance+c.RemoteBalance) * 100
			efficiency = (float64(totalIn) + float64(totalOut)) / float64(c.Capacity) * 100
		}

		var active string = "-"
		if c.Active {
			active = " "
		}

		m := lnwire.NewShortChanIDFromInt(c.ChanId)
		mark := fmt.Sprintf("%7d:%04d:%1d", m.BlockHeight, m.TxIndex, m.TxPosition)

		var monthFee string
		monthFeeSat := decimal.NewFromInt(int64(sum[c.ChanId].Month.FeeMsat)).Div(decimal.NewFromInt(1000))

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
			c.RemotePubkey[:8],
			c.Capacity,
			c.LocalBalance,
			c.RemoteBalance,
			int64(math.Round(ratio)),
			sum[c.ChanId].Day.AmountSatIn,
			sum[c.ChanId].Day.AmountSatOut,
			sum[c.ChanId].Month.AmountSatIn,
			sum[c.ChanId].Month.AmountSatOut,
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

		t.DayAmountSatIn += sum[c.ChanId].Day.AmountSatIn
		t.DayAmountSatOut += sum[c.ChanId].Day.AmountSatOut
		t.MonthAmountSatIn += sum[c.ChanId].Month.AmountSatIn
		t.MonthAmountSatOut += sum[c.ChanId].Month.AmountSatOut

	}
	if t.LocalBalance > 0 {
		t.Ratio = float64(t.LocalBalance) / float64(t.LocalBalance+t.RemoteBalance) * 100
		t.Efficiency = (float64(t.AmountIn) + float64(t.AmountOut)) / float64(t.Capacity) * 100
	}

	// Print total row
	fmt.Printf(line + "\n")
	fmt.Printf(row+"\n",
		len(channels.Channels),
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

func printContracts(contracts *lnrpc.ForwardingHistoryResponse, id uint64) {

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

	sort.SliceStable(contracts.ForwardingEvents, func(i, j int) bool {
		return contracts.ForwardingEvents[i].Timestamp > contracts.ForwardingEvents[j].Timestamp
	})

	for i, c := range contracts.ForwardingEvents {
		if id != 0 && id != c.ChanIdIn && id != c.ChanIdOut {
			continue
		}

		tm := time.Unix(int64(c.Timestamp), 0).Format(time.RFC3339)

		in := lnwire.NewShortChanIDFromInt(c.ChanIdIn)
		out := lnwire.NewShortChanIDFromInt(c.ChanIdOut)

		markIn := fmt.Sprintf("%7d:%04d:%1d", in.BlockHeight, in.TxIndex, in.TxPosition)
		markOut := fmt.Sprintf("%7d:%04d:%1d", out.BlockHeight, out.TxIndex, out.TxPosition)

		fmt.Printf(row+"\n",
			i+1,
			tm,
			c.Timestamp,
			markIn,
			markOut,
			c.AmtIn,
			c.AmtOut,
			c.FeeMsat,
		)
	}
	return
}

func printRespJSON(resp proto.Message) {
	jsonMarshaler := &jsonpb.Marshaler{
		EmitDefaults: true,
		OrigName:     true,
		Indent:       "    ",
	}

	jsonStr, err := jsonMarshaler.MarshalToString(resp)
	if err != nil {
		fmt.Println("unable to decode response: ", err)
		return
	}

	fmt.Println(jsonStr)
}

package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/rodaine/table"
	pb "go.digitalcircle.com.br/dc/netmux/lib/proto/agent"
)

func status() error {
	if opts.Status.Repeat {
		for {
			err := statusOnce()
			if err != nil {
				return err
			}
			time.Sleep(time.Second)
		}
	}
	return statusOnce()
}

func statusOnce() error {
	cli, err := newClient()
	if err != nil {
		return fmt.Errorf("newClient: %w", err)
	}

	var req = &pb.StringMsg{Msg: ""}
	if opts.Status.Zero {
		req = &pb.StringMsg{Msg: "zero"}
	}
	res, err := cli.Status(context.Background(), req)
	if err != nil {
		return err
	}
	printOut("Measure Time: " + time.Now().String() + "\n")
	printOut("Version: " + strings.Replace(res.Version, "\n", " ", 1) + "\n")

	printOut("Config: " + res.Fname + "\n")

	tbl := table.New("CTX", "SVC", "STATUS", "AUTO", "PROTO", "LADDR", "LPORT", "RADDR", "RPORT", "NCONNS", "SENT", "RECV")
	for _, ctx := range res.Contexts {
		var nconns int32 = 0
		var sent int64 = 0
		var recv int64 = 0
		tbl.AddRow(ctx.Name, "---", ctx.Status, "", "", "", "", "", "")
		for _, svc := range ctx.Services {
			nconns = nconns + svc.Nconns
			sent = sent + svc.Sent
			recv = recv + svc.Recv
			tbl.AddRow(ctx.Name,
				svc.Name,
				svc.Status,
				svc.Auto,
				svc.Proto,
				svc.Localaddr,
				svc.Localport,
				svc.Remoteaddr,
				svc.Remoteport,
				fmt.Sprintf("%v", svc.Nconns),
				humanize.Bytes(uint64(svc.Sent)),
				humanize.Bytes(uint64(svc.Recv)))
		}
		tbl.AddRow(ctx.Name, "---", "---", fmt.Sprintf("Conns: %v", nconns), fmt.Sprintf("Sent: %s", humanize.Bytes(uint64(sent))), fmt.Sprintf("Recv: %s", humanize.Bytes(uint64(recv))), "", "", "")

	}

	tbl.Print()
	printOut("=====\n")

	return nil
}

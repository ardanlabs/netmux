package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ardanlabs.com/netmux/app/nx/service"
	"github.com/ardanlabs.com/netmux/business/grpc/clients/agent"
	"github.com/dustin/go-humanize"
	"github.com/gdamore/tcell/v2"
	"github.com/hpcloud/tail"
	"github.com/rivo/tview"
	"github.com/sirupsen/logrus"
)

func renderTable(tb *tview.Table, cfg *service.Netmux) int {
	row := 0

	tb.SetCell(row, 0, tview.NewTableCell("Context").SetExpansion(0).SetAlign(tview.AlignCenter))
	tb.SetCell(row, 1, tview.NewTableCell("Service").SetExpansion(1).SetAlign(tview.AlignCenter))
	tb.SetCell(row, 2, tview.NewTableCell("Endpoint").SetExpansion(1).SetAlign(tview.AlignCenter))
	tb.SetCell(row, 3, tview.NewTableCell("Direction").SetExpansion(1).SetAlign(tview.AlignCenter))
	tb.SetCell(row, 4, tview.NewTableCell("Status").SetExpansion(1).SetAlign(tview.AlignCenter))

	tb.SetCell(row, 5, tview.NewTableCell("NConns").SetAlign(tview.AlignCenter))
	tb.SetCell(row, 6, tview.NewTableCell("Sent").SetAlign(tview.AlignCenter))
	tb.SetCell(row, 7, tview.NewTableCell("Recv").SetAlign(tview.AlignCenter))

	row++

	drawService := func(svcname string, s *service.Service) {
		tb.SetCell(row, 0, tview.NewTableCell(svcname))
		tb.SetCell(row, 1, tview.NewTableCell(s.Name()))
		tb.SetCell(row, 2, tview.NewTableCell(s.Bridge.String()))

		tb.SetCell(row, 3, tview.NewTableCell(s.Bridge.Direction).SetAlign(tview.AlignCenter))
		tb.SetCell(row, 4, tview.NewTableCell(string(s.Status)))
		tb.SetCell(row, 5, tview.NewTableCell(fmt.Sprintf("%v", s.NConns)))
		tb.SetCell(row, 6, tview.NewTableCell(fmt.Sprintf("%v", humanize.Bytes(uint64(s.Sent)))))
		tb.SetCell(row, 7, tview.NewTableCell(fmt.Sprintf("%v", humanize.Bytes(uint64(s.Recv)))))

		s.Sent = 0
		s.Recv = 0
		row++
	}

	drawContext := func(ctx *service.Context) {
		for _, s := range ctx.Services {
			drawService(ctx.Name, s)
		}
	}

	for _, ctx := range cfg.Contexts {
		tb.SetCell(row, 0, tview.NewTableCell("***"))
		tb.SetCell(row, 1, tview.NewTableCell(ctx.Name))
		tb.SetCell(row, 2, tview.NewTableCell(ctx.Url))
		var dir = ""

		tb.SetCell(row, 3, tview.NewTableCell(dir).SetAlign(tview.AlignCenter))
		tb.SetCell(row, 4, tview.NewTableCell(string(ctx.PortForwardStatus)))
		tb.SetCell(row, 5, tview.NewTableCell(fmt.Sprintf("%v", ctx.NConns)))
		tb.SetCell(row, 6, tview.NewTableCell(fmt.Sprintf("%v", humanize.Bytes(uint64(ctx.Sent)))))
		tb.SetCell(row, 7, tview.NewTableCell(fmt.Sprintf("%v", humanize.Bytes(uint64(ctx.Recv)))))

		ctx.Sent = 0
		ctx.Recv = 0
		row++
		drawContext(ctx)
	}
	return row
}

func Run(cli agent.AgentClient, funcs map[string]func() error) error {

	var buf = &bytes.Buffer{}

	chErr := make(chan error)
	var grid *tview.Grid

	go func() {
		t, err := tail.TailFile("/tmp/netmux.log", tail.Config{Follow: true})
		if err != nil {
			chErr <- err
			return
		}
		for line := range t.Lines {
			buf.WriteString(line.Text + "\n")
		}
	}()

	stream, err := cli.Monitor(context.Background(), &agent.Noop{})
	if err != nil {
		return err
	}

	cfg := &service.Netmux{}

	logrus.SetOutput(buf)
	app := tview.NewApplication()
	logs := tview.NewTextArea()
	logs.SetTitle("Logs").SetBorder(true)
	go func() {
		for {
			res, err := stream.Recv()
			if err != nil {
				chErr <- err
				return
			}
			err = json.Unmarshal([]byte(res.Msg), cfg)
			if err != nil {
				chErr <- err
				return
			}

			table := tview.NewTable().SetBorders(true)
			header := tview.NewTextArea()
			header.SetText(fmt.Sprintf("Netmux - Activity Monitor: %s: %s. Running as: %s. Status: %s", strings.TrimSpace(cfg.Semver), strings.TrimSpace(cfg.Version), cfg.Username, cfg.Status), false)
			footer := tview.NewTextArea()
			footer.SetText(fmt.Sprintf("Config: %s - Last update: %s", cfg.SourceFile, time.Now().String()), false)
			if buf.Len() > 0 {
				logs.SetText(logs.GetText()+buf.String(), true)
				buf.Reset()
			}
			sz := renderTable(table, cfg)
			grid = tview.NewGrid()
			grid.SetBackgroundColor(tcell.ColorBlue)
			grid.SetRows(1, sz*2+1, -1, 1)
			grid.
				AddItem(header, 0, 0, 1, 1, 0, 0, false).
				AddItem(table, 1, 0, 1, 1, 0, 0, false).
				AddItem(logs, 2, 0, 1, 1, 0, 0, false).
				AddItem(footer, 3, 0, 1, 1, 0, 0, false)
			app.SetRoot(grid, true)
			app.Draw()

			time.Sleep(time.Second)
		}
	}()

	go func() {
		err := <-chErr
		if err != nil {
			panic(err)
		}
	}()

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		var fn func() error
		switch event.Rune() {
		case 's':
			fn = funcs["start"]
		case 'x':
			fn = funcs["stop"]
		case 'q':
			os.Exit(0)
		default:
			return event
		}
		err := fn()
		if err != nil {
			modal := tview.NewModal().
				SetText(err.Error()).
				AddButtons([]string{"Quit"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					if buttonLabel == "Quit" {
						app.SetRoot(grid, true).Draw()
					}
				})
			if err := app.SetRoot(modal, false).SetFocus(modal).Run(); err != nil {
				panic(err)
			}
		}
		return event
	})
	if err := app.EnableMouse(true).Run(); err != nil {
		return err
	}
	return nil
}

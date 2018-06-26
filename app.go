package main

import (
	"log"
	"github.com/marcusolsson/tui-go"
	"os"
	"github.com/urfave/cli"
	"time"
	"io/ioutil"
	"encoding/json"
	"strconv"
)

type Config struct {
	ID           string
	LocalAddress string    `json:"localAddress"`
	Channels     []Channel `json:"channels"`
}

type Channel struct {
	RemoteAddress string `json:"remoteAddress"`
	ChannelID     string `json:"channelID"`
	buffer        string
}

func main() {
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Value: "./p1.json",
			Usage: "config application",
		},
	}
	app.Action = runApplication
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func runApplication(c *cli.Context) error {
	inputC := make(chan string)
	outputC := make(chan string)
	runUi(&inputC, &outputC)
	configFile, err := ioutil.ReadFile(c.String("config"))
	if err != nil {
		log.Fatal(err)
		return err
	}
	var config Config
	err = json.Unmarshal(configFile, &config)
	if err != nil {
		log.Fatal(err)
		return err
	}
	conn := NewConnection(config.ID, config.LocalAddress, config.Channels, &outputC)
	go func() {
		for {
			input := <-inputC
			if input == "m" {
				conn.SendMarker()
				continue
			}
			n, err := strconv.Atoi(input)
			if err != nil {
				continue
			}
			conn.SendMessage(n)
		}
	}()
	conn.ReceiveMessage()
	return nil
}

func runUi(inputC, outputC *chan string) {
	history := tui.NewVBox()

	historyScroll := tui.NewScrollArea(history)
	historyScroll.SetAutoscrollToBottom(true)

	historyBox := tui.NewVBox(historyScroll)
	historyBox.SetBorder(true)
	historyBox.SetTitle("Application Log")

	input := tui.NewEntry()
	input.SetFocused(true)
	input.SetSizePolicy(tui.Expanding, tui.Maximum)

	inputBox := tui.NewHBox(input)
	inputBox.SetBorder(true)
	inputBox.SetSizePolicy(tui.Expanding, tui.Maximum)
	inputBox.SetTitle("Input Request")

	chat := tui.NewVBox(historyBox, inputBox)
	chat.SetSizePolicy(tui.Expanding, tui.Expanding)

	input.OnSubmit(func(e *tui.Entry) {
		*inputC <- e.Text()
		input.SetText("")
	})

	ui, err := tui.New(chat)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			output := <-*outputC
			ui.Update(func() {
				history.Append(tui.NewHBox(
					tui.NewLabel("["+time.Now().Format("01/02/2006 15:04:05")+"]Receive: "+output),
					tui.NewSpacer(),
				))
			})
		}
	}()

	ui.SetKeybinding("Esc", func() { ui.Quit() })

	go func() {
		if err := ui.Run(); err != nil {
			log.Fatal(err)
		}
	}()
}

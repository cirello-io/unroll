package main

import (
	"context"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("unroll: ")
	appToken := os.Getenv("SLACK_APP_TOKEN")
	token := os.Getenv("SLACK_BOT_TOKEN")
	channels := strings.Split(os.Getenv("SLACK_CHANNELS"), ",")
	client := slack.New(token, slack.OptionDebug(true), slack.OptionLog(log.Default()), slack.OptionAppLevelToken(appToken))
	socketClient := socketmode.New(client, socketmode.OptionDebug(true), socketmode.OptionLog(log.Default()))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case socketEvent := <-socketClient.Events:
				if socketEvent.Type != socketmode.EventTypeEventsAPI {
					continue
				}
				event, ok := socketEvent.Data.(slackevents.EventsAPIEvent)
				if !ok {
					continue
				}
				socketClient.Ack(*socketEvent.Request)
				if event.InnerEvent.Type != "message" {
					continue
				}
				innerEvent, ok := event.InnerEvent.Data.(*slackevents.MessageEvent)
				if !ok {
					continue
				}
				if innerEvent.ThreadTimeStamp == "" {
					continue
				}
				if !slices.Contains(channels, innerEvent.Channel) {
					continue
				}
				permalinkParam := slack.PermalinkParameters{
					Channel: innerEvent.Channel,
					Ts:      innerEvent.TimeStamp,
				}
				url, err := client.GetPermalink(&permalinkParam)
				if err != nil {
					log.Println("cannot get permalink:", err)
					continue
				}
				_, _, err = client.PostMessage(
					innerEvent.Channel,
					slack.MsgOptionAttachments(slack.Attachment{
						Pretext: "<@" + innerEvent.User + "> said:",
						Text:    innerEvent.Text,
						FromURL: url,
						Color:   "#D0D0D0",
						Footer:  "<" + url + "|view in thread>",
					}),
				)
				if err != nil {
					log.Println("cannot copy message:", err)
					continue
				}
			}
		}
	}()
	if err := socketClient.Run(); err != nil {
		log.Fatal(err)
	}
}

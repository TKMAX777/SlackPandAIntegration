package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/TKMAX777/panda"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	scm "github.com/slack-go/slack/socketmode"
)

type SlackHandler struct {
	api  *slack.Client
	scm  *scm.Client
	self *slack.AuthTestResponse

	panda *panda.Handler

	timeSet ReglarFiles

	regexp struct {
		time *regexp.Regexp
	}
}

func NewSlackHandler(token string, panda *panda.Handler, regularFile string) (s *SlackHandler, err error) {
	var api = slack.New(
		token,
		slack.OptionAppLevelToken(Settings.Slack.EventToken),
	)
	var scmAPI = scm.New(api)

	res, err := api.AuthTest()
	if err != nil {
		return
	}

	s = &SlackHandler{
		api: api, scm: scmAPI, self: res, panda: panda,
	}

	s.regexp.time = regexp.MustCompile(`time\s+set\s+(\d?\d):(\d?\d)`)

	err = s.timeSet.Read(regularFile)

	return
}

func (s *SlackHandler) Start() {
	go func() {
		var err = s.scm.Run()
		if err != nil {
			fmt.Println(err)
		}
		return
	}()
	for ev := range s.scm.Events {
		switch ev.Type {
		case scm.EventTypeConnected:
			fmt.Printf("Start websocket connection with Slack\n")
		case scm.EventTypeEventsAPI:
			s.scm.Ack(*ev.Request)

			evp, _ := ev.Data.(slackevents.EventsAPIEvent)
			switch evp.Type {
			case slackevents.CallbackEvent:
				switch evi := evp.InnerEvent.Data.(type) {
				case *slackevents.AppMentionEvent:
					s.messageGet(evi)
				}
			case slackevents.AppRateLimited:
			}
		}
	}
}

func (s *SlackHandler) StartReglarSend(reg ReglarFiles) {
	for _, t := range reg.List {
		var now = time.Now()
		if now.Hour() != t.Time.Hour() ||
			now.Minute() != t.Time.Minute() {
			continue
		}

		Slack.SendAssignments(t.ChannelID)

		time.Sleep(time.Minute)
	}
}

func (s *SlackHandler) messageSend(channelID, text string) (respChannel, respTimestamp string, err error) {
	return s.api.PostMessage(
		channelID,
		slack.MsgOptionAsUser(true),
		slack.MsgOptionText(text, false),
	)
}

func (s *SlackHandler) messageGet(message *slackevents.AppMentionEvent) {
	switch {
	case strings.Contains(message.Text, "課題を確認"):
		s.SendAssignments(message.Channel)
	case strings.Contains(message.Text, "set"):
		s.reglarAdd(message.Text, message.Channel)
	case strings.Contains(message.Text, "remove"):
		s.reglarRemove(message.Text, message.Channel)
	case strings.Contains(message.Text, "regular check"):
		s.reglarListSend(message.Channel)
	default:
		s.messageSend(message.Channel, fmt.Sprintf("<@%s> helpで機能詳細を表示します。", s.self.UserID))
	}
}

func (s *SlackHandler) reglarAdd(text, channelID string) {
	if !s.regexp.time.MatchString(text) {
		return
	}

	for _, ts := range s.regexp.time.FindAllStringSubmatch(text, -1) {
		if len(ts) < 3 {
			continue
		}

		t, err := time.Parse("15:04", ts[1]+":"+ts[2])
		if err != nil {
			continue
		}

		s.timeSet.Add(ReglarFile{t, channelID})
	}

	s.timeSet.Save()

	s.messageSend(channelID, "登録しました。")
	s.reglarListSend(channelID)
}

func (s *SlackHandler) reglarRemove(text, channelID string) {
	var dels []int

	for _, num := range strings.Split(strings.Split(text, "remove")[1], ",") {
		n, err := strconv.Atoi(strings.TrimSpace(num))
		if err != nil {
			continue
		}

		dels = append(dels, n)
	}

	sort.Ints(dels)
	for i := 0; i < len(dels)/2; i++ {
		dels[i], dels[len(dels)-i-1] = dels[len(dels)-i-1], dels[i]
	}

	for _, num := range dels {
		s.timeSet.Remove(num)
	}

	s.timeSet.Save()

	s.reglarListSend(channelID)
}

func (s *SlackHandler) reglarListSend(channelID string) {
	var text string
	text = "次の時間の投稿が予約されています。\n"
	for i, ts := range s.timeSet.List {
		text += fmt.Sprintf("%d %02d:%02d <#%s>\n", i+1, ts.Time.Hour(), ts.Time.Minute(), ts.ChannelID)
	}

	s.messageSend(channelID, text)
}

func (s *SlackHandler) SendAssignments(channelID string) (err error) {
	var text string
	s.messageSend(channelID, "現在PandAに公開されている課題は次の通りです。")

	asss, err := s.panda.AssignmentGet()
	if err != nil {
		return
	}

	var now = time.Now()

	const Day = 3600 * 24

	for _, ass := range asss {
		t, err := time.Parse(time.RFC3339, ass.DueTimeString)
		if err != nil {
			continue
		}

		var emoji string
		switch {
		case t.Unix()-now.Unix() < Day:
			emoji = ":red_circle:"
		case t.Unix()-now.Unix() < 5*Day:
			emoji = ":large_orange_circle:"
		case t.Unix()-now.Unix() < 7*Day:
			emoji = ":large_green_circle:"
		default:
			emoji = ":large_blue_circle:"
		}

		var c = s.panda.ContentGet(ass.Context)

		var subject string
		if len(c) < 1 {
			subject = "科目名不詳"
		} else {
			subject = fmt.Sprintf("<%s|%s>", panda.BaseURI+"/portal/site/"+ass.Context, c[0].Title)
		}

		text += fmt.Sprintf(
			"%s%s\n　%s %s\n",
			emoji,
			subject,
			t.Format("Jan 2(Mon) 15:04"),
			fmt.Sprintf("<%s|%s>", ass.EntityURL, ass.Title),
		)
	}

	s.messageSend(channelID, text)

	return
}

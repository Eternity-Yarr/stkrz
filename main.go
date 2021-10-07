package main

import (
	"encoding/hex"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"sort"
	"stkrz/lib"
	"stkrz/model"
	"stkrz/repo"
	"strings"
)

var bot *tgbotapi.BotAPI

var states = map[int]model.State{}

func main() {
	var err error
	bot, err = tgbotapi.NewBotAPI("-")
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.InlineQuery != nil {
			query := update.InlineQuery
			processInline(query)

		}

		if update.Message != nil {
			msg := update.Message
			if msg.Sticker != nil {
				processSticker(msg.From, *msg.Sticker)
			} else if msg.Chat.ID == int64(msg.From.ID) {
				processPrivate(msg)
			} else {
				log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
				msg.ReplyToMessageID = update.Message.MessageID

				bot.Send(msg)
			}
		}
		continue
	}
}
func processInline(query *tgbotapi.InlineQuery) {
	log.Printf("[%s] %s", query.From.UserName, query.Query)

	stickers := repo.FindByQuery(query.From.ID, query.Query)

	var queryResult []interface{}
	log.Printf("[%s] Found %d results", query.From.UserName, len(stickers))

	var config tgbotapi.InlineConfig

	if len(stickers) > 0 {
		for _, sticker := range stickers {
			queryResult = append(queryResult, lib.AnswerSticker(sticker))
		}
		config = lib.AnswerInline(query.ID, queryResult)
	} else {
		config = lib.AnswerInline(query.ID, []interface{}{})
	}
	encodedQuery := hex.EncodeToString([]byte(query.Query))
	if len(encodedQuery) > 60 {
		config.SwitchPMParameter = "START_PARAM_EMPTY" // ErrorCode:400 Description:Bad Request: START_PARAM_EMPTY Parameters:<nil>}, Bad Request: START_PARAM_EMPTY
	} else {
		config.SwitchPMParameter = encodedQuery
	}
	config.SwitchPMText = "Добавить стикер"
	bot.AnswerInlineQuery(config)
}

func processPrivate(m *tgbotapi.Message) {
	log.Printf("[%s] <- %s", m.From.UserName, m.Text)
	if m.IsCommand() {
		processCommand(m)
		return
	}

	if state, found := states[m.From.ID]; found && state.Type == model.AcceptingTags {
		s, err := repo.FindSticker(state.StickerId, m.From.ID)
		if err != nil {
			sendMessage(m.From, `¯\_(ツ)_/¯`)
			states[m.From.ID] = model.State{Type: model.WaitingSticker}
		} else {
			tags := strings.Split(m.Text, ",")

			newTags := normalizeAndDeduplicate(tags, s.Tags)
			repo.SetTags(state.StickerId, m.From.ID, newTags)

			s.Tags = newTags
			txt := formatAnswer(s)
			sendMessage(m.From, txt)
		}
	} else {
		sendMessage(m.From, "Сначала стикер.")
		states[m.From.ID] = model.State{Type: model.WaitingSticker}
	}
}

func processCommand(m *tgbotapi.Message) {
	switch m.Command() {
	case "done":
		states[m.From.ID] = model.State{Type: model.WaitingSticker}
		sendMessage(m.From, "Ok, давай стикер.")
	case "clear":
		if state, found := states[m.From.ID]; found && state.Type == model.AcceptingTags {
			repo.ClearTags(state.StickerId, m.From.ID)
			sendMessage(m.From, "Ok. Убрал теги.")
		}
	case "start":
		res, err := hex.DecodeString(m.CommandArguments())
		if err == nil {
			sendMessage(m.From, fmt.Sprintf(`Кидай стикер для тега "<code>%s</code>"`, string(res)))
			states[m.From.ID] = model.State{
				Type: model.AcceptingSticker,
				Tag:  string(res),
			}
		} else {
			sendMessage(m.From, "Наверное тэг был слишком длинный, поэтому ничего не вышло. Кидай стикер, потом тег.")
		}
	default:
		sendMessage(m.From, `¯\_(ツ)_/¯`)
	}
}

func processSticker(user *tgbotapi.User, sticker tgbotapi.Sticker) {
	log.Printf("[%s] Got sticker '%s' emoji: %s", user.UserName, sticker.FileID, sticker.Emoji)
	sess := repo.GetSession()
	defer sess.Close()

	var s model.StickerR
	err := repo.GetPersonal(sess).Find(bson.M{"stickerId": sticker.FileID, "userId": user.ID}).One(&s)
	if err == mgo.ErrNotFound {
		id := repo.GetNewId()
		s.UserId = user.ID
		s.RecordId = id
		s.StickerId = sticker.FileID
		if state, found := states[user.ID]; found && state.Type == model.AcceptingSticker {
			s.Tags = []string{state.Tag}
		} else {
			s.Tags = []string{}
		}

		repo.GetPersonal(sess).Upsert(bson.M{"stickerId": sticker.FileID, "userId": user.ID}, s)

		txt := formatAnswer(s)
		states[user.ID] = model.State{Type: model.AcceptingTags, StickerId: s.StickerId}

		sendMessage(user, txt)
	} else if err == nil {

		if state, found := states[user.ID]; found && state.Type == model.AcceptingSticker {

			s.Tags = append(s.Tags, state.Tag)
			repo.SetTags(s.StickerId, user.ID, s.Tags)
			log.Printf("Now the tags for sticker %s is %+v", s.StickerId, s.Tags)
		}

		txt := formatAnswer(s)
		sendMessage(user, txt)

		states[user.ID] = model.State{Type: model.AcceptingTags, StickerId: s.StickerId}

	} else {
		// db err
	}
}

func normalizeAndDeduplicate(tags []string, existingTags []string) (newTags []string) {
	dedup := map[string]bool{}
	for _, t := range append(tags, existingTags...) {
		if t != "" {
			dedup[t] = true
		}
	}
	for t := range dedup {
		tag := strings.ToLower(strings.TrimSpace(t))
		newTags = append(newTags, tag)
	}
	sort.Strings(newTags)
	return newTags
}

func formatAnswer(s model.StickerR) string {
	txt := fmt.Sprintf("Стикер #%d", s.RecordId)
	if len(s.Tags) > 0 {
		txt += "\n<b>Тэги</b>: "
		txt += strings.Join(s.Tags, ", ")
		txt += "\n"
	}
	txt += `
Давай теги, разделяй запятыми.

/done как закончишь
/clear чтоб убрать все теги у стикера`
	return txt
}

func sendMessage(user *tgbotapi.User, txt string) {
	msg := tgbotapi.NewMessage(int64(user.ID), txt)
	msg.ParseMode = "html"
	log.Printf("[%s] -> %s", user.UserName, txt)
	bot.Send(msg)
}

package lib

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"stkrz/model"
)

func AnswerInline(queryId string, queryResult []interface{}) tgbotapi.InlineConfig {
	return tgbotapi.InlineConfig{
		InlineQueryID: queryId,
		Results:       queryResult,
		IsPersonal:    true,
		CacheTime:     1,
	}
}

func AnswerSticker(sticker model.StickerR) tgbotapi.InlineQueryResultCachedSticker {
	return tgbotapi.InlineQueryResultCachedSticker{
		ID:        GenRandomString(16),
		StickerID: sticker.StickerId,
		Type:      "sticker",
	}
}

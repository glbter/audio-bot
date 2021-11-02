package main

import (
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("API_TOKEN"))
	if err != nil {
		log.Fatal("couldn't start bot ", err)
	}

	bot.Debug = true

	updConf := tgbotapi.NewUpdate(0)
	updConf.Timeout = 30
	upds := bot.GetUpdatesChan(updConf)
	for upd := range upds {
		if upd.Message == nil {
			continue
		}

		msg := tgbotapi.NewMessage(upd.Message.Chat.ID, upd.Message.Text)
		msg.ReplyToMessageID = upd.Message.MessageID

		if _, err := bot.Send(msg); err != nil {
			log.Println("failed to send message ", msg)
		}
	}
}

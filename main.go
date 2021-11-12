package main

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"os"
	"strings"
)

const (
	memesOnPage = 9

	prev  = "prev"
	close = "close"
	next  = "next"
)

const (
	songId = "CQACAgIAAxkBAAN3YYpU0neBGgOZwqT4endvfzC8ZtgAAmUFAAJYb5hLpbpSMdocBAMiBA"
)

func main() {
	client := NewClient(os.Getenv("API_MEME"))
	bot, err := tgbotapi.NewBotAPI(os.Getenv("API_TOKEN"))
	if err != nil {
		log.Fatal("couldn't start bot ", err)
	}

	bot.Debug = true

	updConf := tgbotapi.NewUpdate(0)

	updConf.Timeout = 30
	upds, err := bot.GetUpdatesChan(updConf)
	if err != nil {
		log.Fatal("couldn't gt upds", err)
	}

	for upd := range upds {

		if upd.CallbackQuery != nil {
			dt := upd.CallbackQuery.Data
			if strings.Contains(dt, close) {
				bot.DeleteMessage(
					tgbotapi.NewDeleteMessage(
						upd.CallbackQuery.Message.Chat.ID,
						upd.CallbackQuery.Message.MessageID,
					),
				)

				continue
			}

			// go to the previous page
			if strings.Contains(dt, prev) {
				if strings.Split(dt, "|")[1] != "" {
					// delete message and send new message
					bot.AnswerCallbackQuery(tgbotapi.NewCallback(upd.CallbackQuery.ID, prev))
					continue
				}

				bot.AnswerCallbackQuery(tgbotapi.NewCallback(upd.CallbackQuery.ID, "you are on the first page"))
			}

			// go to the next page
			if strings.Contains(dt, next) {
				if strings.Split(dt, "|")[1] != "" {
					bot.AnswerCallbackQuery(tgbotapi.NewCallback(upd.CallbackQuery.ID, next))
					continue
				}

				bot.AnswerCallbackQuery(tgbotapi.NewCallback(upd.CallbackQuery.ID, "you are on the last page"))
			}

			sendVoiceMeme(bot, client, upd.CallbackQuery.Message, dt)
		}

		if upd.Message == nil {
			continue
		}

		//if upd.Message.Audio != nil {
		//	chatId := upd.Message.Chat.ID
		//	fileId := upd.Message.Audio.FileID
		//	msg := tgbotapi.NewAudioShare(chatId, fileId)
		//
		//	msg.ReplyToMessageID = upd.Message.MessageID
		//	if _, err := bot.Send(msg); err != nil {
		//		log.Println("failed to send message ", msg)
		//	}
		//}

		if upd.Message.Voice != nil {
			chatId := upd.Message.Chat.ID
			fileId := upd.Message.Voice.FileID
			msg := tgbotapi.NewVoiceShare(chatId, fileId)

			msg.ReplyToMessageID = upd.Message.MessageID
			if _, err := bot.Send(msg); err != nil {
				log.Println("failed to send message ", msg)
			}
		}

		if upd.Message.Text != "" {
			go func(message *tgbotapi.Message) {
				resp, err := client.FindMeme(message.Text)

				if err != nil {
					// TODO: handle basing on error
					log.Fatalf("failed: text <%v>, err: %v", message.Text, err)
				}

				msg := generateMemesResponse(resp, message.Chat.ID)
				if _, err := bot.Send(msg); err != nil {
					log.Println("failed to send message ", msg)
				}
			}(upd.Message)
		}
	}
}

type Results struct {
	Data []string
	Prev string
	Next string
}

func generateKeyboard(data Results) tgbotapi.InlineKeyboardMarkup {
	rowLen := 3
	row := make([]tgbotapi.InlineKeyboardButton, 0, rowLen)
	board := make([][]tgbotapi.InlineKeyboardButton, 0, 3)

	for i, x := range data.Data {
		if i%rowLen == 0 {
			board = append(board, row)
			row = make([]tgbotapi.InlineKeyboardButton, 0, rowLen)
		}

		//tgbotapi.InlineK

		elem := tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v", i+1), x)
		row = append(row, elem)
	}

	board = append(board, row)
	board = append(
		board,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏪", fmt.Sprintf("%v|%v", prev, data.Prev)),
			tgbotapi.NewInlineKeyboardButtonData("⏹", close),
			tgbotapi.NewInlineKeyboardButtonData("⏩", fmt.Sprintf("%v|%v", next, data.Next)),
		),
	)

	return tgbotapi.NewInlineKeyboardMarkup(board...)
}

func generateList(data []string, page, allAmount int) string {
	var builder strings.Builder
	builder.WriteString(pageToAmount(page, allAmount) + "\n")
	builder.WriteString("Memes \n")
	builder.WriteString("\n")
	for i, x := range data {
		builder.WriteString(fmt.Sprintf("%v. %v \n", i+1, x))
	}

	return builder.String()
}

func pageToAmount(page, allAmount int) string {
	right := page * memesOnPage
	left := (page - 1) * memesOnPage
	return fmt.Sprintf("%v-%v from %v", left, right, allAmount)
}

func generateMemesResponse(resp MemeResponse, chatId int64) tgbotapi.MessageConfig {
	//stubNames := []string{"1","2","3","4","5","6","7","8","9"}
	names := resp.Memes.ToNames()
	txt := generateList(names, 1, resp.Amount)
	msg := tgbotapi.NewMessage(chatId, txt)

	ids := resp.Memes.ToIds()
	msg.ReplyMarkup = generateKeyboard(Results{
		Data: ids,
		Prev: "",
		Next: "",
	})

	return msg
}

func sendVoiceMeme(bot *tgbotapi.BotAPI, client *Client, message *tgbotapi.Message, data string) {
	// get meme from backend
	meme, err := client.GetMeme(data)
	if err != nil {
		// TODO: handle basing on error
		log.Fatalf("failed: meme %v, err: %v", meme, err)
	}

	msg := tgbotapi.NewVoiceShare(message.Chat.ID, meme.FileId)
	if _, err := bot.Send(msg); err != nil {
		log.Println("failed to send message ", msg)
	}
}

//
//func deleteList(bot *tgbotapi.BotAPI, dt string) error{
//	data := strings.Split(dt, "|")
//	chatId, err := strconv.ParseInt(data[2], 10, 10)
//	if err != nil {
//		return err
//	}
//
//	msgId, err := strconv.Atoi(data[1])
//	if err != nil {
//		return err
//	}
//
//	_, err = bot.DeleteMessage(tgbotapi.NewDeleteMessage(chatId, msgId))
//	return err
//}

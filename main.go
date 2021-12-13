package main

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	fileCache map[int64]chan string
	mu        sync.RWMutex
)

const (
	memesOnPage = 9

	prev = "prev"
	clos = "clos"
	next = "next"
)

const (
	songId = "CQACAgIAAxkBAAN3YYpU0neBGgOZwqT4endvfzC8ZtgAAmUFAAJYb5hLpbpSMdocBAMiBA"
)

func main() {
	fileCache = make(map[int64]chan string)
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

			// clos the list with memes
			if strings.Contains(dt, clos) {
				deleteMemesList(bot, upd.CallbackQuery.Message)
				continue
			}

			// go to the previous page
			if strings.Contains(dt, prev) {
				if page := strings.Split(dt, "|")[1]; page != "" {
					moveToPage(upd, bot, client, page)
					bot.AnswerCallbackQuery(tgbotapi.NewCallback(upd.CallbackQuery.ID, prev))
					continue
				}

				bot.AnswerCallbackQuery(tgbotapi.NewCallback(upd.CallbackQuery.ID, "you are on the first page"))
			}

			// go to the next page
			if strings.Contains(dt, next) {
				if page := strings.Split(dt, "|")[1]; page != "" {
					moveToPage(upd, bot, client, page)
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

		if upd.Message.Voice != nil {
			chatId := upd.Message.Chat.ID
			fileId := upd.Message.Voice.FileID
			//msg := tgbotapi.NewVoiceShare(chatId, fileId)
			msg := tgbotapi.NewMessage(chatId, "type a name for audio")

			msg.ReplyToMessageID = upd.Message.MessageID
			if _, err := bot.Send(msg); err != nil {
				log.Println("failed to send message ", msg)
			}

			chName := make(chan string, 1)
			mu.RLock()
			fileCache[chatId] = chName
			mu.RUnlock()

			go func(chatId int64, fileId string) {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
				defer cancel()

				select {
				case <-ctx.Done():
					break
				case memeName := <-chName:
					err := client.AddMeme(Meme{Name: memeName, Id: fileId})
					if err != nil {
						msg := tgbotapi.NewMessage(chatId, "couldn't save your file")
						if _, err := bot.Send(msg); err != nil {
							log.Println("failed to send message ", msg)
						}
					}
				}

				mu.RLock()
				delete(fileCache, chatId)
				mu.RUnlock()
			}(chatId, fileId)
		}

		if txt := upd.Message.Text; txt != "" {
			mu.Lock()
			chName, ok := fileCache[upd.Message.Chat.ID]
			mu.Unlock()
			// send meme to the server
			if ok {
				chName <- txt
				continue
			}

			go func(message *tgbotapi.Message) {
				resp, err := client.FindMeme(message.Text, "1")

				if err != nil {
					// TODO: handle basing on error
					log.Fatalf("failed: text <%v>, err: %v", message.Text, err)
				}

				var nextPage string
				if resp.Amount > memesOnPage {
					nextPage = "2"
				}

				msg := generateMemesResponse(resp, message.Chat.ID, upd.Message.Text, "", nextPage)
				if _, err := bot.Send(msg); err != nil {
					log.Println("failed to send message ", msg)
				}
			}(upd.Message)
		}
	}
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

// clos the list with memes
func deleteMemesList(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	bot.DeleteMessage(
		tgbotapi.NewDeleteMessage(
			message.Chat.ID,
			message.MessageID,
		),
	)
}

func moveToPage(upd tgbotapi.Update, bot *tgbotapi.BotAPI, client *Client, page string) {
	// delete message and send new message
	text := upd.CallbackQuery.Message.Text
	query := strings.Split(text, "\n")[0]
	pageNum, err := strconv.Atoi(page)
	if err != nil {
		log.Fatalf("convert page to int: %v", err)
	}

	resp, err := client.FindMeme(query, page)

	if err != nil {
		// TODO: handle basing on error
		log.Fatalf("failed: text <%v>, err: %v", query, err)
	}

	msg := generateMemesResponse(
		resp,
		upd.CallbackQuery.Message.Chat.ID,
		upd.Message.Text,
		fmt.Sprintf("%v", pageNum-1),
		fmt.Sprintf("%v", pageNum+1),
	)

	deleteMemesList(bot, upd.CallbackQuery.Message)

	if _, err := bot.Send(msg); err != nil {
		log.Println("failed to send message ", msg)
	}
}

//func saveMeme(bot *tgbotapi.BotAPI, client *Client, message *tgbotapi.Message, data string)

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

//-------------------------
// page is starting from 1
//-----------------------
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
	leonidStickerId = "CAACAgIAAxkBAAO5YbumOq1XxQ5lYrpPm0snSCGDKP8AAhAAA7mTYQz6zQERiCwshSME"
)

func main() {
	fileCache = make(map[int64]chan string)
	memeApiUrl := os.Getenv("MEME_API")
	if memeApiUrl == "" {
		log.Fatal("get MEME_API: nil token")
	}

	client := NewClient(memeApiUrl)

	token := os.Getenv("API_TOKEN")
	if token == "" {
		log.Fatal("get API_TOKEN: nil token")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal("couldn't start bot: ", err)
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
				continue
			}

			// go to the next page
			if strings.Contains(dt, next) {
				if page := strings.Split(dt, "|")[1]; page != "" {
					moveToPage(upd, bot, client, page)
					bot.AnswerCallbackQuery(tgbotapi.NewCallback(upd.CallbackQuery.ID, next))
					continue
				}

				bot.AnswerCallbackQuery(tgbotapi.NewCallback(upd.CallbackQuery.ID, "you are on the last page"))
				continue
			}

			sendVoiceMeme(bot, client, upd.CallbackQuery.Message, dt)
			continue
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
						fmt.Println("send voice meme: ", err)
						if _, err := bot.Send(msg); err != nil {
							log.Println("failed to send message ", msg)
						}
					}

					msg := tgbotapi.NewStickerShare(chatId, leonidStickerId)
					if _, err := bot.Send(msg); err != nil {
						log.Println("failed to send message ", msg)
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


type Client struct {
	url string
}

type Meme struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Memes []Meme

func (memes Memes) ToNames() []string {
	names := make([]string, 0, len(memes))
	for _, m := range memes {
		names = append(names, m.Name)
	}

	return names
}

func (memes Memes) ToIds() []string {
	names := make([]string, 0, len(memes))
	for _, m := range memes {
		names = append(names, m.Id)
	}

	return names
}

type MemeResponse struct {
	Memes  Memes
	Amount int
}

type VoiceMeme struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	FileId string `json:"telegramFileId"`
}

// url + AudioFileController/
func NewClient(url string) *Client {
	return &Client{
		url: url+"api/AudioFile/",
	}
}

// find memes in backend by providing query
func (c *Client) FindMeme(params string, page string) (MemeResponse, error) {
	var memes Memes

	take := 10
	p, err := strconv.Atoi(page)
	if err != nil {
		return MemeResponse{}, fmt.Errorf("convert page to int: %w", err)
	}
	skip := (p - 1) * take

	uri := fmt.Sprintf("%vGetByQuery?take=%v&query=%v&skip=%v", c.url, take, url.QueryEscape(params), skip)

	fmt.Println(uri)

	resp, err := http.Get(uri) //TODO: specify url
	if err != nil {
		return MemeResponse{}, fmt.Errorf("find meme: get resource: %w", err)
	}

	fmt.Println(resp)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return MemeResponse{}, fmt.Errorf("find meme: read body: %w", err)
	}

	fmt.Println(body)

	if err := json.Unmarshal(body, &memes); err != nil {
		return MemeResponse{}, fmt.Errorf("find meme: unmarshal: %w", err)
	}

	res := MemeResponse{Memes: memes, Amount: len(memes)}
	return res, nil
}

func (c *Client) GetMeme(id string) (VoiceMeme, error) {
	fmt.Println(id)
	fmt.Println(c.url + "GetById?id=" + id)
	resp, err := http.Get(c.url + "GetById?id=" + id) //TODO: specify url
	if err != nil {
		return VoiceMeme{}, fmt.Errorf("get meme: get resource: %w", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return VoiceMeme{}, fmt.Errorf("get meme: read body: %w", err)
	}

	var memeResp VoiceMeme
	if err := json.Unmarshal(body, &memeResp); err != nil {
		return VoiceMeme{}, fmt.Errorf("get meme: unmarshal: %w, %s", err, body)
	}

	return memeResp, nil
}

type NewMeme struct {
	Id   string `json:"telegramFileId"`
	Name string `json:"name"`
}

func (c *Client) AddMeme(m Meme) error {

	meme := NewMeme{Id: m.Id, Name: m.Name}
	body, err := json.Marshal(meme)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}

	fmt.Println(m)
	fmt.Println(body)
	fmt.Println(c.url+"Create")

	resp, err := http.Post(c.url+"Create", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("post request: %w", err)
	}
	defer resp.Body.Close()

	fmt.Println(resp)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("create meme error")
	}

	return nil
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
			tgbotapi.NewInlineKeyboardButtonData("⏹", clos),
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

func generateMemesResponse(resp MemeResponse, chatId int64, query, prevPage, nextPage string) tgbotapi.MessageConfig {
	//stubNames := []string{"1","2","3","4","5","6","7","8","9"}
	names := resp.Memes.ToNames()

	txt := generateList(names, 1, resp.Amount)
	msg := tgbotapi.NewMessage(chatId, query+"\n"+txt)

	ids := resp.Memes.ToIds()
	msg.ReplyMarkup = generateKeyboard(Results{
		Data: ids,
		Prev: prevPage,
		Next: nextPage,
	})

	return msg
}

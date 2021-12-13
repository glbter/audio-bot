package main

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"strings"
)

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

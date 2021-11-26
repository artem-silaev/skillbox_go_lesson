package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type wallet map[string]float64

var db = map[int64]wallet{}

type binanceResp struct {
	Price float64 `json:"price,string"`
	Code  int64   `json:"code"`
}

func main() {
	bot, err := tgbotapi.NewBotAPI("2126965659:AAFlM4gALry_IS796r429JvyM9B4q0M_oP4")
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		msgArr := strings.Split(update.Message.Text, " ")
		summ := 0.0
		switch msgArr[0] {
		case "ADD":
			summ, err = strconv.ParseFloat(msgArr[2], 64)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Не верный ввод"))
				continue
			}
			if _, ok := db[update.Message.Chat.ID]; !ok {
				db[update.Message.Chat.ID] = wallet{}
			}
			db[update.Message.Chat.ID][msgArr[1]] += summ
			msg := fmt.Sprintf("Баланс: %s %f", msgArr[1], db[update.Message.Chat.ID][msgArr[1]])
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
		case "SUB":
			summ, err = strconv.ParseFloat(msgArr[2], 64)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Не верный ввод"))
				continue
			}
			if _, ok := db[update.Message.Chat.ID]; !ok {
				db[update.Message.Chat.ID] = wallet{}
			}
			db[update.Message.Chat.ID][msgArr[1]] -= summ
			msg := fmt.Sprintf("Баланс: %s %f", msgArr[1], db[update.Message.Chat.ID][msgArr[1]])
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
		case "DEL":
			delete(db[update.Message.Chat.ID], msgArr[1])
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Валюта удалена"))
		case "SHOW":
			msg := ""
			var usdSumm float64
			rubPrice, err := getPrice("USD", "RUB")
			if err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
			}
			for key, value := range db[update.Message.Chat.ID] {
				coinPrice, err := getPrice(key, "USD")
				if err != nil {
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
				}
				msg += fmt.Sprintf("В долларах %s: %f [%.2f]\n", key, value, value*coinPrice)
				msg += fmt.Sprintf("В рублях %s: %f [%.2f]\n", key, value, value*coinPrice*rubPrice)
				usdSumm += value * coinPrice
			}
			msg += fmt.Sprintf("Общая сумма %.2f\n", usdSumm)
			msg += fmt.Sprintf("Общая сумма в рублях %.2f", usdSumm*rubPrice)
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
		default:
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Не верный ввод"))
		}
	}
}

func getPrice(coin string, toCoin string) (price float64, err error) {
	resp, err := http.Get(fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%sT%s", coin, toCoin))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var jsonResp binanceResp
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	if err != nil {
		return
	}

	if jsonResp.Code != 0 {
		err = errors.New("Некорректная валюта")
		return
	}

	price = jsonResp.Price

	return
}

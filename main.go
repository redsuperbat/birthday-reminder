package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron"
)

type Birthday struct {
	Name string `json:"name"`
	Date string `json:"date"`
}

type BirthdayConfig struct {
	Birthdays []Birthday `json:"birthdays"`
}

type Creds struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type Token struct {
	Token string `json:"token"`
}

func getCreds(connectionString string) (*Creds, string, error) {
	// Remove the protocol part from the connection string
	parts := strings.SplitN(connectionString, "://", 2)
	if len(parts) < 2 {
		return nil, "", errors.New("invalid conn string")
	}
	connectionString = parts[1]

	// Split the remaining string into username, password, host, and port
	parts = strings.Split(connectionString, "@")
	if len(parts) < 2 {
		return nil, "", errors.New("invalid conn string")
	}
	credentials := parts[0]
	url := parts[1]

	// Split the credentials part into username and password
	parts = strings.Split(credentials, ":")
	if len(parts) < 2 {
		return nil, "", errors.New("invalid conn string")
	}
	creds := Creds{
		Username: parts[0],
		Password: parts[1],
	}

	return &creds, url, nil
}

func getTelegramChatId() int64 {
	chatIdStr := os.Getenv("TELEGRAM_CHAT_ID")
	if chatIdStr == "" {
		log.Fatalln("No chat id was provided")
	}
	chatId, err := strconv.Atoi(chatIdStr)
	if err != nil {
		log.Fatalln("Invalid chat id provided, expected number got", chatIdStr)
	}
	return int64(chatId)
}

func main() {
	creds, url, err := getCreds(os.Getenv("RSB_CONFIG_URL"))
	if err != nil {
		log.Panicln(err)
	}
	chatId := getTelegramChatId()
	client := resty.New()
	botToken := os.Getenv("TELEGRAM_BOT_KEY")

	url = "http://" + url
	token := Token{}
	resp, err := client.R().SetBody(creds).SetResult(&token).Post(url + "/auth/login")
	if resp.StatusCode() >= 400 {
		log.Panicln(resp)
	}
	log.Println(token)
	if err != nil {
		log.Panicln(err)
	}
	birthdays := BirthdayConfig{}
	resp, err = client.R().SetAuthToken(token.Token).SetResult(&birthdays).Get(url + "/api/config/birthdays.json")
	if resp.StatusCode() >= 400 {
		log.Panicln(resp)
	}
	if err != nil {
		log.Panicln(err)
	}
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panicln(err)
	}
	log.Println("received birthdays to watch")
	log.Println(birthdays)

	c := cron.New()
	err = c.AddFunc("0 0 9 * * *", func() {
		today := time.Now().Format("2006-01-02")
		log.Println("checking birthdays...")

		var birthdayPeeps []Birthday
		for _, v := range birthdays.Birthdays {
			if v.Date[4:] != today[4:] {
				continue
			}
			birthdayPeeps = append(birthdayPeeps, v)
		}

		if len(birthdayPeeps) == 0 {
			log.Println("No ones birthday today 😢")
			return
		}

		for _, v := range birthdayPeeps {
			log.Printf("it's %s birthday today", v.Name)
			log.Println("notifying max...")
			text := fmt.Sprintf("Hey, its %s's birthday today!\nDon't forget to message them and say Happy Birthday!", v.Name)
			msg := tgbotapi.NewMessage(chatId, text)
			log.Println("sending message...")
			if _, err := bot.Send(msg); err != nil {
				log.Printf("Unable to send message because of %v", err.Error())
			}
			log.Println("message sent!")
		}
	})

	c.Start()
	if err != nil {
		log.Panicln(err)
	}
	select {}

}

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	tb "github.com/go-telegram-bot-api/telegram-bot-api"
)

const yaSpeller string = "https://speller.yandex.net/services/spellservice.json/checkText"

type spelledJSON struct {
	Word    string   `json:"word"`
	Spelled []string `json:"s"`
}

type counted struct {
	Word  string
	Times int
}

func findInSlice(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}

func getSpell(url string, text string) string {
	var words []spelledJSON
	text = fmt.Sprintf("text=%s&lang=&options=&format=", text)
	body := strings.NewReader(text)

	resp, err := http.Post(url, "application/x-www-form-urlencoded", body)
	checkErr(err)
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	checkErr(err)
	json.Unmarshal(data, &words)

	var processedWords []string
	var result string

	if len(words) > 0 {
		result = "Возможные ошибки и опечатки:\n"
		for _, i := range words {
			var wordGuess string
			if findInSlice(processedWords, i.Word) {
				continue
			}
			processedWords = append(processedWords, i.Word)
			for _, j := range i.Spelled {
				wordGuess = wordGuess + fmt.Sprintf("%s? ", j)
			}
			result = result + fmt.Sprintf("%s – %s\n", i.Word, wordGuess)
		}
	}
	return result
}

func checkErr(e error) {
	if e != nil {
		panic(e)
	}
}

func getFormattedLine(arr []string) string {
	wastedChars := regexp.MustCompile(`\.|,|"|\(|\)|\?|\!`)
	hyphen := regexp.MustCompile(`([а-я])-([а-я])`)

	line := strings.Join(arr, " ")
	line = strings.ToLower(line)
	line = wastedChars.ReplaceAllString(line, "")
	line = hyphen.ReplaceAllString(line, "$1 $2")

	return line
}

func countWord(arr []string) string {
	var sorted []counted
	var result string

	m := make(map[string]int)

	line := getFormattedLine(arr)

	for _, item := range strings.Fields(line) {
		m[item]++
	}

	for k, v := range m {
		sorted = append(sorted, counted{k, v})
	}

	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Times > sorted[j].Times })

	header := "Часто встречающиеся слова и тире:\n"
	if len(sorted) > 0 {
		for _, kv := range sorted {
			if kv.Times > 1 {
				result = fmt.Sprintf("%s%s: %d, ", result, kv.Word, kv.Times)
			}
		}
	}
	if result != "" {
		result = strings.TrimSuffix(result, ", ")
		result = header + result

	}
	return result
}

func main() {
	bot, err := tb.NewBotAPI(os.Getenv("APITOKEN"))
	checkErr(err)

	ch := tb.NewUpdate(0)
	ch.Timeout = 60

	updates, _ := bot.GetUpdatesChan(ch)

	time.Sleep(time.Millisecond * 500)
	updates.Clear()

	for update := range updates {
		if update.UpdateID >= ch.Offset {
			ch.Offset = update.UpdateID + 1
		}

		var respToUser string
		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				respToUser = "Давай текст до 10000 знаков, поглядим"
			}
		} else {
			spelled := getSpell(yaSpeller, update.Message.Text)
			commonWords := countWord(strings.Fields(update.Message.Text))
			if spelled != "" {
				respToUser = respToUser + spelled
			}
			if commonWords != "" {
				respToUser = respToUser + commonWords
			}
			if respToUser == "" {
				respToUser = "всё норм"
			}
		}
		msg := tb.NewMessage(update.Message.Chat.ID, respToUser)
		bot.Send(msg)
	}
}

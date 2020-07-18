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
			case "faq":
				respToUser = `Q: Что это?
				A: Это бот, который проверяет орфографию. Сделано на базе технологии Яндекса, которая называется Спеллер. 
				В качестве приятного бонуса – считает дубликаты слов в переданном тексте.
				Q: И зачем?
				A: В основном, для проверки себя и контроля за тем, что ты пишешь. Можно скормить текст презентации, или стихотворение, или что угодно ещё.
				Q: А как пользоваться?
				A: Просто отправляешь текст, и всё. 
				Q: А пунктуацию проверяет?
				A: К сожалению, пока что нет. Со временем, надеюсь, начнёт. 
				Q: Какие ещё ограничения?
				A: Основное – объём текста не должен превышать 10000 знаков.
				Q: А перспективы?
				A: Хочется верить, что радужные, но "темна вода во облацех", как говорил Мерлин. Из основного: хочется прикрутить проверку пунктуации и 
				проверку на сервисе Главреда – glvrd.ru. 
				Q: И что, прям все ошибки находит?
				A: К сожалению, нет. Это очень во многом зависит от базы, на которой обучается механизм Яндекса, и там встречаются пробелы. Из конкретных "детских болезней" 
				собственной логики, пожалуй, основное – то, что он не учитывает словоформы. Одно и то же слово в разных падежах будет восприниматься ботом как два разных.
				Q: Я нашёл баг. Куда писать?
				A: Непосредственно разработчику – @apetropolsky.
				Q: А если есть идея и хочу ей поделиться?
				A: Тогда туда же.
				Q: А если сам хочу сделать?
				A: Тогда вот репозиторий на гитхабе, можно сделать пулл-реквест: https://github.com/apetropolsky/spellcheckerbot. Если что-то с ним пойдёт не так – снова пиши разработчику, попробуем разобраться. 
				Q: Как узнать о новых фишках?
				A: Думаю, что будет команда для запроса чейндж-лога, но см. раздел про перспективы. 
				`
			case "credits":
				respToUser = "@apetropolsky"
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
				respToUser = "Выглядит нормально"
			}
		}
		msg := tb.NewMessage(update.Message.Chat.ID, respToUser)
		bot.Send(msg)
	}
}

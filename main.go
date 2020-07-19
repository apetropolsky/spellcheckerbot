package main

import (
	"bufio"
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
	Guesses []string `json:"s"`
}

type counted struct {
	Word  string
	Times int
}

func notEmptyString(s string) bool {
	if len(s) > 0 {
		return false
	}
	return true
}

func checkErr(e error) {
	if e != nil {
		panic(e)
	}
}

func findInSlice(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}

func readFile(path string) []string {
	var lines []string
	file, err := os.OpenFile(path, os.O_RDWR, 0755)
	checkErr(err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
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
			for _, j := range i.Guesses {
				wordGuess = wordGuess + fmt.Sprintf("%s? ", j)
			}
			result = result + fmt.Sprintf("%s – %s\n", i.Word, wordGuess)
		}
	}
	return result
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
		var replyToUser string

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				replyToUser = "Давай текст до 10000 знаков, поглядим"
			case "faq":
				faq := readFile("/var/spellchecker/faq")
				replyToUser = strings.Join(faq, "\n")
			case "credits":
				replyToUser = "@apetropolsky"
			}
		} else {
			spelled := getSpell(yaSpeller, update.Message.Text)
			commonWords := countWord(strings.Fields(update.Message.Text))
			if notEmptyString(spelled) {
				replyToUser = replyToUser + spelled
			}
			if notEmptyString(commonWords) {
				replyToUser = replyToUser + commonWords
			}
			if notEmptyString(replyToUser) {
				replyToUser = "Выглядит нормально"
			}
		}
		msg := tb.NewMessage(update.Message.Chat.ID, replyToUser)
		bot.Send(msg)
	}
}

package httpserver

import (
	"Bingo/bingo"
	"Bingo/config"
	"Bingo/random"
	"Bingo/webhub"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	hub      *webhub.Hub
	password string
)

func Listen() {

	password = random.RandSeq(8)
	log.Info("Password: " + password)

	hub = webhub.NewHub()
	go hub.Run()

	http.HandleFunc("/bingo/", handleBoard)
	http.HandleFunc("/main/", handleMain)
	http.HandleFunc("/completed/", handleCompleted)
	http.HandleFunc("/reroll/", handleReroll)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		webhub.ServeWs(hub, w, r)
	})
	http.Handle("/", http.FileServer(http.Dir("frontend")))

	http.ListenAndServe(":8080", nil)
}

func handleCompleted(resp http.ResponseWriter, req *http.Request) {
	if req.URL.Query().Get("pass") != password {
		return
	}

	url := strings.Split(req.URL.Path, "/")
	bingolink := url[2]
	word := url[3]
	word = strings.TrimSpace(word)

	bingo := bingo.Bingos[bingolink]

	newValue := false

	for _, field := range bingo.Words {
		field = strings.TrimSpace(field)
		if field == word {
			newValue = !bingo.Completed[field]
			bingo.Completed[field] = newValue
			break
		}
	}

	hub.Broadcast <- []byte(word + ";" + strconv.FormatBool(newValue))
	bingo.Store(config.Json.StoragePath)
}

func handleReroll(resp http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		return
	}

	url := strings.Split(req.URL.Path, "/")
	if len(url) < 4 {
		return
	}

	bingolink := url[3]
	boardlink := url[4]

	submittedPass := req.URL.Query().Get("pass")
	oldWord := req.URL.Query().Get("value")

	bingo, exists := bingo.Bingos[bingolink]
	if !exists {
		return
	}
	board, exists := bingo.Boards[boardlink]
	if !exists {
		return
	}

	if submittedPass != board.Password {
		return
	}

	if board.Rerolls <= 0 {
		return
	}

	possibleWords := copyArrayFromMap(bingo.Completed)
	for word, val := range bingo.Completed {
		if val {
			index := findIndex(possibleWords, word)
			if index >= 0 {
				possibleWords = remove(possibleWords, index)
			}
		}
	}
	for _, word := range board.Content {
		index := findIndex(possibleWords, word)
		if index >= 0 {
			possibleWords = remove(possibleWords, index)
		}
	}

	if len(possibleWords) <= 0 {
		return
	}

	newWord := possibleWords[rand.Intn(len(possibleWords))]

	index := findIndex(board.Content, oldWord)

	board.Content[index] = newWord
	board.Rerolls -= 1

	hub.Broadcast <- []byte("Reroll")
	bingo.Store(config.Json.StoragePath)
	resp.Header().Add("content-type", "text/plain")
	resp.Write([]byte(newWord + ";" + strconv.Itoa(board.Rerolls)))
}

func handleMain(resp http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		return
	}

	url := strings.Split(req.URL.Path, "/")
	bingolink := url[2]

	bingo, exists := bingo.Bingos[bingolink]
	if !exists {
		return
	}

	body := ""
	for _, field := range bingo.Words {
		field = strings.TrimSpace(field)
		value := bingolink + "/" + field
		if bingo.Completed[field] {
			body += `<button onclick="onClick(this)" class="button-completed" value="` + value + `" id="` + field + `">` + field + "</button>"
		} else {
			body += `<button onclick="onClick(this)" class="button" value="` + value + `" id="` + field + `">` + field + "</button>"
		}
	}

	htmlTemplate, err := ioutil.ReadFile("frontend/index.html")
	if err != nil {
		return
	}

	html := strings.ReplaceAll(string(htmlTemplate), "{{body}}", body)

	resp.Write([]byte(html))
}

func handleBoard(resp http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		return
	}

	url := strings.Split(req.URL.Path, "/")
	bingolink := url[2]
	boardlink := url[3]

	bingo, exists := bingo.Bingos[bingolink]
	if !exists {
		return
	}
	board, exists := bingo.Boards[boardlink]
	if !exists {
		return
	}

	body := ""
	for _, field := range board.Content {
		field = strings.TrimSpace(field)
		if bingo.Completed[field] {
			body += `<div class="grid-item-completed" id="` + field + `">` + field + "</div>"
		} else {
			body += `<div class="grid-item" id="` + field + `" onclick="reroll(this)">` + field + "</div>"
		}
	}

	miniboards := ""

	playernames := ""

	count := 0
	for _, otherBoard := range bingo.Boards {
		if otherBoard.Id == board.Id {
			continue
		}
		playernames += `<p class="playername">` + otherBoard.UserName + `</p>`

		miniboards += `<div class="grid-container-mini">`
		for _, field := range otherBoard.Content {
			field = strings.TrimSpace(field)
			if bingo.Completed[field] {
				miniboards += `<div class="grid-item-completed-mini" id="` + strconv.Itoa(count) + "/" + field + `">` + field + "</div>"
			} else {
				miniboards += `<div class="grid-item-mini" id="` + strconv.Itoa(count) + "/" + field + `">` + field + "</div>"
			}
		}

		count++
		miniboards += `</div>`
	}

	htmlTemplate, err := ioutil.ReadFile("frontend/board.html")
	if err != nil {
		return
	}

	html := strings.ReplaceAll(string(htmlTemplate), "{{board}}", body)
	html = strings.ReplaceAll(html, "{{miniboards}}", miniboards)
	html = strings.ReplaceAll(html, "{{playernames}}", playernames)
	html = strings.ReplaceAll(html, "{{rerolls}}", strconv.Itoa(board.Rerolls))

	resp.Write([]byte(html))
}

func findIndex(array []string, val string) int {
	for i, s := range array {
		if val == s {
			return i
		}
	}
	return -1
}

// func copyArray(array []string) []string {
// 	newArray := make([]string, len(array))
// 	for i, s := range array {
// 		newArray[i] = s
// 	}
// 	return newArray
// }

func copyArrayFromMap(oldMap map[string]bool) []string {
	newArray := make([]string, len(oldMap))
	i := 0
	for key := range oldMap {
		newArray[i] = key
		i++
	}
	return newArray
}

func remove(s []string, i int) []string {
	s[i] = s[len(s)-1]
	// We do not need to put s[i] at the end, as it will be discarded anyway
	return s[:len(s)-1]
}

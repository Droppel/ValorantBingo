package main

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type Config struct {
	StoragePath string `json:"storagePath"`
	LogLevel    string `json:"logLevel"`
}

var (
	bingos   map[string]*Bingo
	hub      *Hub
	config   Config
	password string
)

func main() {
	var err error
	config, err = readConfig()
	if err != nil {
		log.WithError(err).Error("Failed to parse the config")
		return
	}

	logLevel, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		log.WithError(err).Error("Could not parse the loglevel")
		return
	}

	log.SetLevel(logLevel)

	password = randSeq(8)
	log.Info("Password: " + password)

	bingos = make(map[string]*Bingo)
	rand.Seed(time.Now().UnixNano())

	go InitBot()

	hub = newHub()
	go hub.run()

	http.HandleFunc("/bingo/", handleBoard)
	http.HandleFunc("/main/", handleMain)
	http.HandleFunc("/completed/", handleCompleted)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	http.Handle("/", http.FileServer(http.Dir("frontend")))

	http.ListenAndServe(":8080", nil)
}

func AddBingo(bin *Bingo) {

	id := bin.Id

	bingos[id] = bin

	return
}

func handleCompleted(resp http.ResponseWriter, req *http.Request) {
	if req.URL.Query().Get("pass") != password {
		return
	}

	url := strings.Split(req.URL.Path, "/")
	bingolink := url[2]
	word := url[3]
	word = strings.TrimSpace(word)

	bingo := bingos[bingolink]

	newValue := false

	for _, field := range bingo.Words {
		field = strings.TrimSpace(field)
		if field == word {
			newValue = !bingo.Completed[field]
			bingo.Completed[field] = newValue
			break
		}
	}

	hub.broadcast <- []byte(word + ";" + strconv.FormatBool(newValue))
	bingo.Store()
}

func handleMain(resp http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		return
	}

	url := strings.Split(req.URL.Path, "/")
	bingolink := url[2]

	bingo, exists := bingos[bingolink]
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

	bingo, exists := bingos[bingolink]
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
			body += `<div class="grid-item" id="` + field + `">` + field + "</div>"
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

	resp.Write([]byte(html))
}

func readConfig() (Config, error) {

	configFile, err := ioutil.ReadFile("config.json")
	if err != nil {
		return Config{}, err
	}

	config := &Config{}
	err = json.Unmarshal(configFile, config)
	if err != nil {
		return Config{}, err
	}

	return *config, nil
}

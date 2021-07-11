package main

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"strings"
)

type Bingo struct {
	Kind      string                `json:"kind"`
	Words     []string              `json:"words"`
	Completed map[string]bool       `json:""`
	Wordsize  int                   `json:"wordsize"`
	Size      int                   `json:"size"`
	Boards    map[string]BingoBoard `json:"boards"`
	Id        string                `json:"id"`
}

type BingoBoard struct {
	Content  []string `json:"content"`
	Id       string   `json:"id"`
	UserName string   `json:"username"`
}

type Field struct {
	Content   string `json:"content"`
	Completed bool   `json:"completed"`
}

func Create(_kind string, _size int) (*Bingo, error) {
	bin := Bingo{Kind: _kind, Size: _size, Id: randSeq(16), Boards: make(map[string]BingoBoard)}

	wordsFile, err := ioutil.ReadFile("bingos/" + _kind + ".txt")
	if err != nil {
		return nil, err
	}

	words := strings.Split(string(wordsFile), "\n")
	bin.Completed = make(map[string]bool)
	bin.Words = make([]string, 0)

	for _, word := range words {
		word = strings.TrimSpace(word)
		if word[0] == '#' {
			continue
		}

		bin.Words = append(bin.Words, word)
		bin.Completed[word] = false
	}

	bin.Wordsize = len(bin.Words)

	bin.Store()
	return &bin, nil
}

func (bin *Bingo) CreateBoard(id string, username string) BingoBoard {
	existingBoard, exists := bin.Boards[id]
	if exists {
		return existingBoard
	}

	board := BingoBoard{}
	board.Content = make([]string, 0, bin.Size)
	for i := 0; i < bin.Size; i++ {
		randomField := bin.Words[rand.Intn(bin.Wordsize)]
		for contains(board.Content, randomField) {
			randomField = bin.Words[rand.Intn(bin.Wordsize)]
		}
		board.Content = append(board.Content, randomField)
	}

	board.Id = id
	board.UserName = username

	bin.Boards[board.Id] = board

	bin.Store()
	return board
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func contains(array []string, val string) bool {

	for _, cont := range array {
		if cont == val {
			return true
		}
	}

	return false
}

func (b *Bingo) Store() error {
	path := config.StoragPath

	jsonBingo, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}

	ioutil.WriteFile(path+b.Kind+"_"+b.Id+".json", jsonBingo, 0644)

	return nil
}

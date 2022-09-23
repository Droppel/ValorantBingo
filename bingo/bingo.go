package bingo

import (
	"Bingo/config"
	"Bingo/random"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"strings"
)

type Bingo struct {
	Kind      string                 `json:"kind"`
	Words     []string               `json:"words"`
	Completed map[string]bool        `json:""`
	Wordsize  int                    `json:"wordsize"`
	Size      int                    `json:"size"`
	Boards    map[string]*BingoBoard `json:"boards"`
	Id        string                 `json:"id"`
	OwnerId   string                 `json:"ownerID"`
	GuildId   string                 `json:"guildID`
	Password  string                 `json:"password"`
}

type BingoBoard struct {
	Content  []string `json:"content"`
	Id       string   `json:"id"`
	UserName string   `json:"username"`
	Rerolls  int      `json:"rerolls"`
	Password string   `json:"password"`
}

type Field struct {
	Content   string `json:"content"`
	Completed bool   `json:"completed"`
}

var (
	Bingos map[string]*Bingo
)

func (b *Bingo) CheckFinished() []*BingoBoard {
	finishedBoards := make([]*BingoBoard, 0)
boardLoop:
	for _, board := range b.Boards {
		//Create done array
		done := make([]bool, 25)
		for k, cont := range board.Content {
			done[k] = b.Completed[cont]
		}

		//Check Rows
		for r := 0; r < 5; r++ {
			if done[0+r*5] && done[1+r*5] && done[2+r*5] && done[3+r*5] && done[4+r*5] {
				finishedBoards = append(finishedBoards, board)
				continue boardLoop
			}
		}
		//Check Columns
		for c := 0; c < 5; c++ {
			if done[0+c] && done[5+c] && done[2*5+c] && done[3*5+c] && done[4*5+c] {
				finishedBoards = append(finishedBoards, board)
				continue boardLoop
			}
		}
		//Check Diagonals
		if done[0] && done[6] && done[12] && done[18] && done[24] {
			finishedBoards = append(finishedBoards, board)
			continue boardLoop
		}
		if done[4] && done[8] && done[12] && done[16] && done[20] {
			finishedBoards = append(finishedBoards, board)
			continue boardLoop
		}
	}

	return finishedBoards
}

func AddBingo(bin *Bingo) {
	id := bin.Id

	Bingos[id] = bin
}

func Create(guildId, ownerId string, _kind string, _size int) (*Bingo, error) {
	bin := Bingo{
		OwnerId:  ownerId,
		GuildId:  guildId,
		Kind:     _kind,
		Size:     _size,
		Id:       random.RandSeq(16),
		Boards:   make(map[string]*BingoBoard),
		Password: random.RandSeq(8),
	}

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

	bin.Store(config.Json.StoragePath)
	return &bin, nil
}

func (bin *Bingo) CreateBoard(id string, username string, totalRerolls int) *BingoBoard {
	existingBoard, exists := bin.Boards[id]
	if exists {
		return existingBoard
	}

	board := &BingoBoard{}
	board.Password = random.RandSeq(8)
	board.Rerolls = totalRerolls

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

	bin.Store(config.Json.StoragePath)
	return board
}

func contains(array []string, val string) bool {

	for _, cont := range array {
		if cont == val {
			return true
		}
	}

	return false
}

func (b *Bingo) Store(path string) error {
	jsonBingo, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}

	ioutil.WriteFile(path+b.Kind+"_"+b.Id+".json", jsonBingo, 0644)

	return nil
}

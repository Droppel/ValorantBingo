package bingo

import (
	"Bingo/config"
	"testing"
)

func TestCreateBoard(t *testing.T) {
	bin, err := Create("12345", "valorant", 5)

	if err != nil {
		t.Fail()
	}
	bin.CreateBoard("reandomid", "user", config.Json.GameSettings.TotalRerolls)
}

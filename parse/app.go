package parse

import (
	"mtg_test/mtgadata"
)

type GameState struct {
	game                 string
	match                string
	playerID             string
	intendToJoinGameWith string
	playerDecks          string
	draftHistory         string
	lastBlob             string
	errorCount           int
	collection           string
	allCards             *mtgadata.MtgaData
}

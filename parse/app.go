package parse

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
}

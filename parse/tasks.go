package parse

import (
	"bufio"
	"fmt"
	"io"
	"mtg_test/mtgadata"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	simplejson "github.com/bitly/go-simplejson"
)

const timeLayout = "1/2/2006 3:04:05 PM"

func TailFileTask(fileName string, fromstart bool, follow bool, linesChan chan string) {
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Print("Couldn't open file ", fileName)
		return
	}
	if !fromstart {
		file.Seek(0, io.SeekEnd)
	}
	buf := bufio.NewReader(file)
	for {
		where, _ := file.Seek(0, io.SeekCurrent)
		line, isPartial, err := buf.ReadLine()
		if isPartial {
			fmt.Print("Line too long")
			break
		}
		if err == io.EOF {
			if !follow {
				return
			}
			file.Seek(where, io.SeekStart)
			time.Sleep(100 * time.Millisecond)
		} else {
			linesChan <- string(line)
		}
	}
}

func FrameBlocksTask(linesChan chan string, blocksChan chan []string) {
	currentBlock := []string{}
	for {
		line := <-linesChan
		if len(line) == 0 {
			continue
		}
		if strings.HasPrefix(line, "[UnityCrossThreadLogger]") || strings.HasPrefix(line, "[Client GRE]") {
			// this is the start of a new block (with title), end the last one
			currentBlock = []string{line}
		} else if len(currentBlock) > 0 {
			// 	# we're in the middle of a block somewhere
			currentBlock = append(currentBlock, line)
			if strings.HasPrefix(line, "]") || strings.HasPrefix(line, "}") {
				// 	# this is the END of a block, end it and start a new one
				blocksChan <- currentBlock
				currentBlock = []string{}
			}
		}
	}
}

type BlockData struct {
	timestamp          time.Time
	logLine            int
	blockTitle         string
	blockTitleSeq      int
	requestNotResponse bool
	jsonStr            string
}

func (d BlockData) String() string {
	return "timestamp:" + d.timestamp.Format(timeLayout) + ", Title:" + d.blockTitle + ", JSON:" + d.jsonStr
}

type words []string

func (l words) fromEnd(n int) string {
	return l[len(l)-n]
}

func ProcessBlocksTask(blocksChan chan []string, jsonChan chan BlockData) {
	headerRe := regexp.MustCompile(`^\[.+\](.+(PM|AM))(.*)$`)
	sourceRe := regexp.MustCompile(`^.+ (.+)\(([0-9]+)\)`)
	for {
		blockLines := <-blocksChan

		if len(blockLines) < 2 {
			continue
		}

		meta := BlockData{}

		headerMatches := headerRe.FindStringSubmatch(blockLines[0])
		if headerMatches == nil {
			fmt.Println("Invalid Header", blockLines[0])
			continue
		}
		timeLayout := "1/2/2006 3:04:05 PM"
		timestamp, err := time.Parse(timeLayout, headerMatches[1])
		if err != nil {
			fmt.Println("Invalid Timestamp", blockLines[0])
			continue
		}
		meta.timestamp = timestamp
		headerExtra := headerMatches[2]
		var extraHeaderWords words = strings.Split(headerExtra, " ")
		titleLine := blockLines[1]

		if strings.HasPrefix(blockLines[1], "==>") || strings.HasPrefix(blockLines[1], "<==") {
			/*
				these logs looks like:

				[UnityCrossThreadLogger]6/7/2018 7:21:03 PM
				==> Log.Info(530):
				{
					"json": "stuff"
				}
			*/
			titleMatches := sourceRe.FindStringSubmatch(titleLine)
			if titleMatches == nil {
				fmt.Println("Invalid Source", blockLines[1])
				continue
			}
			meta.blockTitle = titleMatches[1]
			blockTitleSeq, err := strconv.Atoi(titleMatches[2])
			if err != nil {
				fmt.Println("Invalid blockTitleSeq", blockLines[1])
				continue
			}
			meta.blockTitleSeq = blockTitleSeq
			meta.requestNotResponse = strings.HasPrefix(titleLine, "==>")

			meta.jsonStr = strings.Join(blockLines[2:], "\n")
			if strings.HasPrefix(meta.jsonStr, "[") {
				// this is not valid json, we need to surround it with a header such that it's an object instead of a list
				meta.jsonStr = `{"` + meta.blockTitle + `":` + meta.jsonStr + `}`
			}
		} else if strings.Contains(titleLine, "{") {
			/*
				these logs look like:

				[UnityCrossThreadLogger]6/7/2018 7:21:03 PM: Match to 26848417E29213FE: GreToClientEvent
				{
					"json": "stuff"
				}
			*/
			meta.blockTitle = extraHeaderWords.fromEnd(1)
			meta.jsonStr = strings.Join(blockLines[1:], "\n")
		} else if strings.HasSuffix(headerExtra, "{") {
			/*
				these blocks looks like:

				[UnityCrossThreadLogger]7/2/2018 10:27:59 PM (-1) Incoming Rank.Updated {
					"json": "stuff
				}
			*/
			meta.blockTitle = extraHeaderWords.fromEnd(2)
			meta.jsonStr = "{" + strings.Join(blockLines[1:], "\n")
		}
		if len(meta.jsonStr) > 0 {
			jsonChan <- meta
		}
	}
}

func checkForClientID(game *GameState, blob *simplejson.Json) {
	if data, ok := (*blob).Get("authenticateResponse").CheckGet("clientId"); ok {
		clientID := data.MustString()
		// screw it, no one else is going to use this message, mess up the timestamp, who cares
		if game.playerID != clientID {
			game.playerID = clientID
			//general_output_queue.put({"authenticateResponse": blob["authenticateResponse"]})
		}
	}
}

func JsonReaderTask(jsonChan chan BlockData, allCards *mtgadata.MtgaData) {
	lastBlob := ""
	gameState := new(GameState)
	gameState.allCards = allCards
	lastDecklist := ""
	// errorCount := 0
	for {
		meta := <-jsonChan
		jsonRecieved, err := simplejson.NewJson([]byte(meta.jsonStr))
		if err != nil {
			fmt.Println("Invalid JSON", meta)
			continue
		}

		if lastBlob == meta.jsonStr {
			continue // don't double fire
		}
		lastBlob = meta.jsonStr

		// check for decklist changes
		if gameState.playerDecks != lastDecklist {
			lastDecklist = gameState.playerDecks
			//decklist_change_queue.put({k: v.to_serializable(transform_to_counted=True) for k, v in lastDecklist.items()})
		}

		// check for gamestate changes
		// hero_library_hash := -1
		// opponent_hand_hash := -1
		// if gameState.game {
		// 	hero_library_hash = hash(mtga_watch_app.game.hero.library)
		// 	opponent_hand_hash = hash(mtga_watch_app.game.opponent.hand)
		// }

		checkForClientID(gameState, jsonRecieved)
		DispatchBlob(&meta, gameState, jsonRecieved)
		//         mtga_watch_app.lastBlob = jsonRecieved
		//         errorCount = 0

		//         hero_library_hash_post = -1
		//         opponent_hand_hash_post = -1
		//         if mtga_watch_app.game:
		//             hero_library_hash_post = hash(mtga_watch_app.game.hero.library)
		//             opponent_hand_hash_post = hash(mtga_watch_app.game.opponent.hand)
		//             if hero_library_hash != hero_library_hash_post or opponent_hand_hash != opponent_hand_hash_post:
		//                 game_state_change_queue.put(mtga_watch_app.game.game_state())  # TODO: BREAKPOINT HERE
		//             if mtga_watch_app.game.final:
		//                 game_state_change_queue.put({"match_complete": True, "gameID": mtga_watch_app.game.match_id})
		//     except:
		//         import traceback
		//         exc = traceback.format_exc()
		//         stack = traceback.format_stack()
		//         mtga_logger.error("{}Exception @ count {}".format(util.ld(True), mtga_watch_app.errorCount))
		//         mtga_logger.error(exc)
		//         mtga_logger.error(stack)
		//         mtga_watch_app.send_error("Exception during check game state. Check log for more details")
		//         if errorCount > 5:
		//             mtga_logger.error("{}error count too high; exiting".format(util.ld()))
		//             return
	}
}

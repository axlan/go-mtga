package main

import (
	mtgadb "mtg_test/mtgadata"
	parse "mtg_test/parse"
)

func main() {

	linesChan := make(chan string, 4)
	blocksChan := make(chan []string, 4)
	jsonChan := make(chan parse.BlockData, 4)
	const testFile = `C:\Users\jdiamond\go\src\mtg_test\output_log.txt`
	//const testFile = `C:\Users\jdiamond\go\src\mtg_test\test.txt

	allCards := mtgadb.LoadMTGAData("mtga.db")

	go parse.FrameBlocksTask(linesChan, blocksChan)
	go parse.ProcessBlocksTask(blocksChan, jsonChan)
	go parse.JsonReaderTask(jsonChan, allCards)

	parse.TailFileTask(testFile, true, false, linesChan)

	// go tailFile(testFile, true, true, linesChan)
	// var input string
	// fmt.Scanln(&input)

}

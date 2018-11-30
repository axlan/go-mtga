package mtgadata

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type Card struct {
	OwnerSeatID string
	MtgaID      string
	Name        string
	Rarity      string
}

func (c Card) String() string {
	return fmt.Sprintf("%v (%v)", c.Name, c.Rarity)
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func column2Slice(rows *sql.Rows) []string {
	var val string
	list := make([]string, 0)
	for rows.Next() {
		err := rows.Scan(&val)
		checkErr(err)
		list = append(list, val)
	}
	return list
}

func enumTable2Slice(table string, col string, db *sql.DB) []string {
	rows, err := db.Query("SELECT " + col + " FROM " + table)
	checkErr(err)
	defer rows.Close()
	return column2Slice(rows)
}

type MtgaData struct {
	db       *sql.DB
	types    []string
	rarities []string
}

func LoadMTGAData(dbFile string) *MtgaData {
	db, err := sql.Open("sqlite3", dbFile)
	checkErr(err)
	rarities := enumTable2Slice("rarity", "rarity_name", db)
	types := enumTable2Slice("types", "type_name", db)
	return &MtgaData{db, types, rarities}
}

func (data MtgaData) GetCard(mtgaID int) *Card {
	rows, err := data.db.Query("SELECT name, card_rarity FROM cards WHERE mtga_id = ?", mtgaID)
	checkErr(err)
	var name string
	var rarityID int
	rows.Next()
	err = rows.Scan(&name, &rarityID)
	checkErr(err)
	card := Card{"", string(mtgaID), name, data.rarities[rarityID]}
	rows.Close()
	return &card
}

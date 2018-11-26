package parse

import (
	"fmt"

	simplejson "github.com/bitly/go-simplejson"
)

type Card struct {
	ownerSeatID string
	mtgaID      string
}

type Pool struct {
	poolName string
	cards    []Card
}

type CardMap map[string]Card

func (p Pool) TotalCount() int {
	return len(p.cards)
}

func (p Pool) CountCardsOwnedBy(seat string) int {
	total := 0
	for _, card := range p.cards {
		if card.ownerSeatID == seat {
			total++
		}
	}
	return total
}

func (p Pool) Count(mtgaID string) int {
	total := 0
	for _, card := range p.cards {
		if card.mtgaID == mtgaID {
			total++
		}
	}
	return total
}

func (p Pool) GroupCards() map[string]int {
	grouped := make(map[string]int)
	for _, card := range p.cards {
		if group, ok := grouped[card.mtgaID]; ok {
			grouped[card.mtgaID] = group + 1
		} else {
			grouped[card.mtgaID] = 1
		}
	}
	return grouped
}

type Deck struct {
	Pool
	deckID string
}

func CreateDeck(name string, id string) *Deck {
	cards := [0]Card{}
	deck := Deck{Pool: Pool{name, cards[:]}, deckID: id}
	return &deck
}

/*


    def transfer_all_to(self, other_pool):
        for card in self.cards:
            other_pool.cards.append(card)
        while self.cards:
            self.cards.pop()

    def transfer_cards_to(self, cards, other_pool):
        # TODO: make this "session safe" (ie if we error on the third card, we should not have transferred the first 2)
        for card in cards:
            self.transfer_card_to(card, other_pool)

    def transfer_card_to(self, card, other_pool):
        # TODO: make this atomic, somehow?
        res = card
        if not isinstance(card, mcard.Card):  # allow you to pass in cards or ids or searches
            res = self.find_one(card)
        self.cards.remove(res)
        other_pool.cards.append(res)

    @classmethod
    def from_sets(cls, pool_name, sets):
        cards = []
        for set in sets:
            for card in set.cards_in_set:
                cards.append(card)
        return Pool(pool_name, cards)

    def find_one(self, id_or_keyword):
        result = set(self.search(id_or_keyword))
        if len(result) < 1:
            raise ValueError("Pool does not contain {}".format(id_or_keyword))
        elif len(result) > 1:
            raise ValueError("Pool search '{}' not narrow enough, got: {}".format(id_or_keyword, result))
        return result.pop()

    def search(self, id_or_keyword, direct_match_returns_single=False):
        keyword_as_int = None
        keyword_as_str = str(id_or_keyword)
        try:
            keyword_as_int = int(id_or_keyword)
            if keyword_as_int < 10000:
                keyword_as_int = None
        except (ValueError, TypeError):
            pass
        results = []
        for card in self.cards:
            if keyword_as_int == card.mtga_id or keyword_as_int == card.set_number:
                return [card]

            keyword_clean = re.sub('[^0-9a-zA-Z_]', '', keyword_as_str.lower())
            if keyword_clean == card.name and direct_match_returns_single:
                return [card]
            if keyword_clean in card.name:
                results.append(card)
        return results

class Deck(Pool):

    def __init__(self, pool_name, deck_id):
        super().__init__(pool_name)
        self.deck_id = deck_id

    def generate_library(self, owner_id=-1):
        library = Library(self.pool_name, self.deck_id, owner_id, -1)
        for card in self.cards:
            game_card = mcard.GameCard(card.name, card.pretty_name, card.cost, card.color_identity, card.card_type,
                                       card.sub_types, card.set, card.rarity, card.set_number, card.mtga_id, owner_id, -1)
            library.cards.append(game_card)
        return library

    def to_serializable(self, transform_to_counted=False):
        obj = {
            "deck_id": self.deck_id,
            "pool_name": self.pool_name,
        }
        if transform_to_counted:
            card_dict = {}
            for card in self.cards:
                card_dict[card.mtga_id] = card_dict.get(card.mtga_id, card.to_serializable())
                card_dict[card.mtga_id]["count_in_deck"] = card_dict[card.mtga_id].get("count_in_deck", 0) + 1
            obj["cards"] = [v for v in card_dict.values()]
            obj["cards"].sort(key=lambda x: x["count_in_deck"])
            obj["cards"].reverse()
            obj["cards"] = obj["cards"]
        else:
            obj["cards"] = [c.to_serializable() for c in self.cards]
        return obj

    def to_min_json(self):
        min_deck = {}
        for card in self.cards:
            if card.mtga_id not in min_deck:
                min_deck[card.mtga_id] = 0
            min_deck[card.mtga_id] += 1
        return {"deckID": self.deck_id, "poolName": self.pool_name, "cards": min_deck}

    @classmethod
    def from_dict(cls, obj):
        deck = Deck(obj["pool_name"], obj["deck_id"])
        for card in obj["cards"]:
            deck.cards.append(mcard.Card.from_dict(card))
        return deck
*/

func ProcessDeck(deckJson *simplejson.Json, saveDeck bool) string {
	deckID, err := deckJson.Get("id").String()
	if err != nil {
		fmt.Printf("Missing deckID")
		return ""
	}
	deckName, err := deckJson.Get("name").String()
	if err != nil {
		fmt.Printf("Missing deckName")
		return ""
	}
	deck := CreateDeck(deckName, deckID)

	fmt.Println(deck.poolName)

	cardObjs, err := deckJson.Get("mainDeck").Array()
	if err != nil {
		fmt.Printf("Missing card array")
		return ""
	}
	numCards := len(cardObjs)
	for i := 0; i < numCards; i++ {
		/*
			why? jsonrpc methods use capitalized instead of lowercase. idk, see for yourself:
			== > DirectGame.Challenge(42):
			{
			    "jsonrpc": "2.0",
			    "method": "DirectGame.Challenge",
			    "params": {
			        "opponentDisplayName": "MTGATracker#78028",
			        "avatar": "Sarkhan_M19_01",
			        "deck": "{\"id\":\"c4d5e085-65c4-4873-9aaf-d9d081bde8e4\",\"name\":\"MTGA DOWN, PANIC
			                    mk 02\",\"format\":\"Standard\",\"description\":\"\",
			                    \"localDescription\":\"Temp string\",
			                    \"deckTileId\":0,\"isValid\":true,\"lastUpdated\":\"2018-09-28T08:35:17.0184205\",
			                    \"mainDeck\":[{\"Id\":67015,\"Quantity\":60},{\"Id\":67017,\"Quantity\":190}],\
			                    \"sideboard\":[]}"
			    },
			    "id": "42"
			}
		*/
		var quantity int
		var id string
		card := deckJson.Get("mainDeck").GetIndex(i)
		if val, err := card.Get("id").String(); err == nil {
			id = val
		} else if val, err := card.Get("Id").String(); err == nil {
			id = val
		} else {
			fmt.Printf("Missing card id")
			return ""
		}
		if val, err := card.Get("quantity").Int(); err == nil {
			quantity = val
		} else if val, err := card.Get("Quantity").Int(); err == nil {
			quantity = val
		} else {
			fmt.Printf("Missing card quantity")
			return ""
		}
		for j := 0; j < quantity; j++ {
			deck.cards = append(deck.cards, Card{"", id})
		}

	}
	fmt.Println(deck.cards)
	return ""
}

/*
def process_deck(deck_dict, save_deck=True):
    for card_obj in deck_dict["mainDeck"]:
        try:
            id_key = "id" if "id" in card_obj else "Id"
            qt_key = "quantity" if "quantity" in card_obj else "Quantity"
            # why? jsonrpc methods use capitalized instead of lowercase. idk, see for yourself:
            # == > DirectGame.Challenge(42):
            # {
            #     "jsonrpc": "2.0",
            #     "method": "DirectGame.Challenge",
            #     "params": {
            #         "opponentDisplayName": "MTGATracker#78028",
            #         "avatar": "Sarkhan_M19_01",
            #         "deck": "{\"id\":\"c4d5e085-65c4-4873-9aaf-d9d081bde8e4\",\"name\":\"MTGA DOWN, PANIC
            #                     mk 02\",\"format\":\"Standard\",\"description\":\"\",
            #                     \"localDescription\":\"Temp string\",
            #                     \"deckTileId\":0,\"isValid\":true,\"lastUpdated\":\"2018-09-28T08:35:17.0184205\",
            #                     \"mainDeck\":[{\"Id\":67015,\"Quantity\":60},{\"Id\":67017,\"Quantity\":190}],\
            #                     \"sideboard\":[]}"
            #     },
            #     "id": "42"
            # }
            card = all_mtga_cards.search(card_obj[id_key])[0]
            for i in range(card_obj[qt_key]):
                deck.cards.append(card)
        except Exception as e:
            mtga_app.mtga_logger.error("{}Unknown mtga_id: {}".format(ld(), card_obj))
            mtga_app.mtga_watch_app.send_error("Could not process deck {}: Unknown mtga_id: {}".format(deck_dict["name"], card_obj))
    if save_deck:
        with mtga_app.mtga_watch_app.game_lock:
            mtga_app.mtga_watch_app.player_decks[deck_id] = deck
            mtga_app.mtga_logger.info("{}deck {} is being saved".format(ld(), deck_dict["name"]))
            mtga_app.mtga_watch_app.save_settings()
    return deck
*/

func ParseEventDecksubmit(game *GameState, blob *simplejson.Json) {
	courseDeck := blob.Get("CourseDeck")
	deck := ProcessDeck(courseDeck, false)
	fmt.Println(deck)
	// mtga_app.mtga_watch_app.intend_to_join_game_with = deck
}

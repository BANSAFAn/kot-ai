package steam

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/ztrue/tracerr"
)

const (
	steamAPIURL = "https://api.steampowered.com/IPlayerService/GetOwnedGames/v1/"
)

// Game представляет информацию об игре
type Game struct {
	AppID     int    `json:"appid"`
	Name      string `json:"name"`
	Playtime2 int    `json:"playtime_2weeks"`
	Playtime  int    `json:"playtime_forever"`
}

// Response представляет ответ от Steam API
type Response struct {
	Response struct {
		Count int    `json:"game_count"`
		Games []Game `json:"games"`
	} `json:"response"`
}

// GetGames возвращает список игр Steam
func GetGames() ([]string, error) {
	apiKey := os.Getenv("STEAM_API_KEY")
	if apiKey == "" {
		return nil, tracerr.New("Переменная окружения STEAM_API_KEY не установлена")
	}

	steamID := os.Getenv("STEAM_ID")
	if steamID == "" {
		return nil, tracerr.New("Переменная окружения STEAM_ID не установлена")
	}

	url := fmt.Sprintf("%s?key=%s&steamid=%s&format=json&include_appinfo=1", steamAPIURL, apiKey, steamID)

	resp, err := http.Get(url)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, tracerr.New(fmt.Sprintf("Ошибка Steam API: %s", resp.Status))
	}

	var steamResponse Response
	if err := json.NewDecoder(resp.Body).Decode(&steamResponse); err != nil {
		return nil, tracerr.Wrap(err)
	}

	var gameNames []string
	for _, game := range steamResponse.Response.Games {
		gameNames = append(gameNames, game.Name)
	}

	return gameNames, nil
}
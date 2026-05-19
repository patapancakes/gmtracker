/*
	gmtracker - the garry's mod 12 server browser
	Copyright (C) 2024  patapancakes <patapancakes@pagefault.games>

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU Affero General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU Affero General Public License for more details.

	You should have received a copy of the GNU Affero General Public License
	along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

var (
	cache Cache
	key   string

	//go:embed templates
	templates      embed.FS
	TemplatesFS, _ = fs.Sub(templates, "templates")

	tmpl = template.Must(template.New("list.html").Funcs(template.FuncMap{"region": region, "platform": platform}).ParseFS(TemplatesFS, "list.html"))
)

type Cache struct {
	LastUpdate time.Time
	Servers    []ServerInfo
}

type GetServerListResponse struct {
	Response struct {
		Servers []ServerInfo `json:"servers"`
	} `json:"response"`
}

type ServerInfo struct {
	Address       string `json:"addr"`
	GamePort      int    `json:"gameport"`
	SteamID       string `json:"steamid"`
	Name          string `json:"name"`
	AppID         int    `json:"appid"`
	GameDirectory string `json:"gamedir"`
	Version       string `json:"version"`
	Product       string `json:"product"`
	Region        int    `json:"region"`
	Players       int    `json:"players"`
	MaxPlayers    int    `json:"max_players"`
	Bots          int    `json:"bots"`
	Map           string `json:"map"`
	Secure        bool   `json:"secure"`
	Dedicated     bool   `json:"dedicated"`
	OS            string `json:"os"`
	GameType      string `json:"gametype"`
}

func main() {
	apikey := flag.String("apikey", "", "steam web api key")
	proto := flag.String("proto", "tcp", "proto for web server")
	addr := flag.String("addr", "127.0.0.1:80", "address for web server")
	flag.Parse()

	if *apikey == "" {
		log.Fatal("an api key is required! get one at https://steamcommunity.com/dev/apikey")
	}

	key = *apikey

	http.HandleFunc("/", handle)

	if *proto == "unix" {
		err := os.Remove(*addr)
		if err != nil && !os.IsNotExist(err) {
			log.Fatalf("failed to delete unix socket: %s", err)
		}
	}

	l, err := net.Listen(*proto, *addr)
	if err != nil {
		log.Fatalf("failed to create web server listener: %s", err)
	}

	defer l.Close()

	if *proto == "unix" {
		err = os.Chmod(*addr, 0777)
		if err != nil {
			log.Fatalf("failed to set unix socket permissions: %s", err)
		}
	}

	http.Serve(l, nil)
}

func handle(w http.ResponseWriter, r *http.Request) {
	si, err := update()
	if err != nil {
		log.Printf("update failed: %s", err)
	}

	err = tmpl.Execute(w, si)
	if err != nil {
		log.Printf("response generation failed: %s", err)
		http.Error(w, "failed to generate response!", http.StatusInternalServerError)
	}
}

func update() (Cache, error) {
	if time.Now().Before(cache.LastUpdate.Add(time.Minute)) {
		return cache, nil
	}

	v := make(url.Values)

	v.Add("filter", `\appid\4000\version_match\1.*`)
	v.Add("key", key)

	r, err := http.Get(fmt.Sprintf("https://api.steampowered.com/IGameServersService/GetServerList/v1/?%s", v.Encode()))
	if err != nil {
		return cache, err
	}

	var response GetServerListResponse
	err = json.NewDecoder(r.Body).Decode(&response)
	if err != nil {
		return cache, err
	}

	cache.LastUpdate = time.Now()
	cache.Servers = response.Response.Servers

	return cache, nil
}

func region(region int) string {
	switch region {
	case 0:
		return "US - East"
	case 1:
		return "US - West"
	case 2:
		return "South America"
	case 3:
		return "Europe"
	case 4:
		return "Asia"
	case 5:
		return "Australia"
	case 6:
		return "Middle East"
	case 7:
		return "Africa"
	}

	return "World"
}

func platform(platform string) string {
	switch platform {
	case "w":
		return "Windows"
	case "m":
		return "Mac"
	case "l":
		return "Linux"
	}

	return "Other"
}

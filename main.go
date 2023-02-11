package main

import (
	"encoding/json"
	"fmt"
	"github.com/alitto/pond"
	"github.com/blackrosezy/go-takwim-solat-malaysia/config"
	"github.com/blackrosezy/go-takwim-solat-malaysia/solat"
	"io/ioutil"
	"sync"
)

var m sync.Mutex

func main() {
	webparser := &solat.WebParser{}
	html, _ := webparser.GetRawData()
	country, err := webparser.Parse(html)
	if err != nil {
		fmt.Println("Error parsing html:", err)
		return
	}
	b, _ := json.MarshalIndent(country, "", "  ")
	fmt.Println(string(b))
	codes := webparser.GetCodes(country.States)

	_ = ioutil.WriteFile("zones.json", b, 0644)

	c, err := config.GetConfig("config.json")
	if err != nil {
		fmt.Println(err)
		return
	}
	a := &solat.Solat{BaseURL: c.BaseURL}
	prayerTimesByZone := make(map[string][]solat.PrayerTime)

	pool := pond.New(10, 1000)

	// Submit 1000 tasks
	for _, code := range codes {
		code := code
		pool.Submit(func() {
			prayerTimes := a.GetWaktuSolatByZone(code)
			m.Lock()
			prayerTimesByZone[code] = prayerTimes
			m.Unlock()
		})
	}

	// Stop the pool and wait for all submitted tasks to complete
	pool.StopAndWait()

	file, _ := json.MarshalIndent(prayerTimesByZone, "", " ")
	_ = ioutil.WriteFile("solat.json", file, 0644)
}

package solat

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type Solat struct {
	BaseURL string
}

func (s *Solat) GetWaktuSolatByZone(zone string) []PrayerTime {
	url := fmt.Sprintf("%s&period=year&zone=%s", s.BaseURL, zone)
	fmt.Println(url)

	timeout := time.Duration(5 * time.Second)
	client := &http.Client{
		Timeout: timeout,
	}
	// Make the API request
	resp, err := client.Get(url)
	defer resp.Body.Close()
	if err != nil {
		fmt.Println(err)
		return []PrayerTime{}
	}

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return []PrayerTime{}
	}

	// Unmarshal the JSON data
	var takwimSolat TakwimSolat
	err = json.Unmarshal(body, &takwimSolat)
	if err != nil {
		fmt.Println(err)
		return []PrayerTime{}
	}

	return takwimSolat.PrayerTimes
}

package solat

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type StateWrapper struct {
	State *State `json:"state"`
}

type State struct {
	Name  string  `json:"name"`
	Zones []Zone `json:"zones"`
}

type Zone struct {
	Code string `json:"code"`
	Places string `json:"places"`
}

type Country struct {
	States []State `json:"states"`
}

type WebParser struct{}

func (wp *WebParser) GetRawData() (string, error) {
	timeout := time.Duration(10 * time.Second)
	client := &http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get("https://www.e-solat.gov.my/")
	if err != nil {
		panic(err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (wp *WebParser) Parse(html string) (Country, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return Country{}, err
	}

	selectEl := doc.Find("select#inputzone")
	if selectEl.Length() == 0 {
		return Country{}, fmt.Errorf("no select element with id 'inputzone' found")
	}

	states, err := wp.parseStates(selectEl)
	if err != nil {
		return Country{}, err
	}

	country := Country{
		States: []State{},
	}

	country.States = states

	return country, nil
}

func (wp *WebParser) GetCodes(states []State) []string {
	var codes []string
	for _, state := range states {
		for _, zone := range state.Zones {
			codes = append(codes, zone.Code)
		}
	}
	return codes
}

func (wp *WebParser) parseStates(selectEl *goquery.Selection) ([]State, error) {
	var states []State
	var parseErr error
	selectEl.Find("optgroup").Each(func(i int, s *goquery.Selection) {
		state := State{
			Name:  s.AttrOr("label", ""),
			Zones: []Zone{},
		}
		zones, err := wp.parseZones(s)
		if err != nil {
			parseErr = err
			return
		}
		state.Zones = zones
		states = append(states, state)
	})
	return states, parseErr
}



func (wp *WebParser) parseZones(s *goquery.Selection) ([]Zone, error) {
	var zones []Zone
	var err error
	s.Find("option").Each(func(j int, t *goquery.Selection) {
		code := t.AttrOr("value", "")
		name, cleanErr := wp.cleanName(t.Text(), code)
		if cleanErr != nil {
			err = fmt.Errorf("failed to clean name %s: %v", t.Text(), cleanErr)
			return
		}
		zone := Zone{
			Code: code,
			Places: name,
		}
		zones = append(zones, zone)
	})
	if err != nil {
		return nil, err
	}
	return zones, nil
}

func (wp *WebParser) cleanName(name string, code string) (string, error) {
	cleanedName := strings.Replace(strings.TrimSpace(name), code+" - ", "", -1)
	return cleanedName, nil
}

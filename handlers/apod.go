package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	en "url-collector/env"
)

const baseExternalAPI = "https://api.nasa.gov/planetary/apod?"

type ApodPayload struct {
	Copyright       string `json:"copyright"`
	Date            string `json:"date"`
	Explanation     string `json:"explanation"`
	Hdurl           string `json:"hdurl"`
	Media_type      string `json:"media_type"`
	Service_version string `json:"service_version"`
	Title           string `json:"title"`
	Url             string `json:"url"`
}

type InMemoryPicturesUrlStore struct {
	urls map[time.Time]string
}

func NewPictureUrlServer() *PictureUrlServer {
	str := InMemoryPicturesUrlStore{urls: make(map[time.Time]string)}
	return &PictureUrlServer{store: str}
}

func (s InMemoryPicturesUrlStore) GetPictureUrlByDate(from, to string) (res []string, err error) {
	const dateLayout = "2006-01-02"
	var picsMap sync.Map
	var dates []string

	fmt.Println("got here")
	s.urls = make(map[time.Time]string)
	startDate, err := time.Parse(dateLayout, from)
	if err != nil {
		return nil, fmt.Errorf("time.Parse('2006-01-02',%s)=%v", from, err)
	}
	endDate, err := time.Parse(dateLayout, to)
	if err != nil {
		return nil, fmt.Errorf("time.Parse('2006-01-02',%s)=%v", to, err)
	}

	days := int(endDate.Sub(startDate).Hours() / 24)

	for i := 0; i < days+1; i++ {
		dt := startDate.AddDate(0, 0, i)
		dates = append(dates, dt.Format(dateLayout))
	}
	wg := sync.WaitGroup{}
	// wg.Add(len(dates))
	for i := 0; i < len(dates); i++ {
		wg.Add(1)
		go func(idx int) {
			fmt.Println(dates[idx])
			pic, e := GetPicturesFromApod(dates[idx])
			if e.Error != "" {
				fmt.Println(e.Error)
				panic(e)
			}
			picsMap.Store(dates[idx], pic.Url)
			wg.Done()
		}(i)
	}
	wg.Wait()
	for i := 0; i < len(dates); i++ {
		url, ok := picsMap.Load(dates[i])
		if !ok {
			return nil, fmt.Errorf("picsMap.Load(dates[%d])=false", i)
		}
		dt, err := time.Parse(dateLayout, dates[i])
		if err != nil {
			return nil, fmt.Errorf("time.Parse('2006-01-02',%s)=%v", dates[i], err)
		}
		s.urls[dt] = fmt.Sprint(url)
		res = append(res, fmt.Sprint(url))
	}
	return res, nil
}
func GetPicturesFromApod(date string) (pld *ApodPayload, e ErrStr) {
	var client http.Client

	apiKey := en.GetEnvVar("API_KEY", "DEMO_KEY")

	fullURL := fmt.Sprintf("%s&api_key=%s&date=%s", baseExternalAPI, apiKey, date)
	fmt.Println(fullURL)
	resp, err := client.Get(fullURL)
	if err != nil {
		e = ErrStr{Error: fmt.Sprintf("client.Get(%s)=%s\n", fullURL, err.Error())}
		return nil, e
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		e = ErrStr{Error: "resp.StatusCode=" + strconv.Itoa(resp.StatusCode) + " for URL: " + fullURL}
		return nil, e
	}
	remainingLimit, err := strconv.Atoi(resp.Header.Get("X-RateLimit-Remaining"))
	if err != nil {
		e = ErrStr{Error: "Cannot get X-RateLimit-Remaining header: " + err.Error()}
		return nil, e
	}
	if remainingLimit < 1 {
		e = ErrStr{Error: "X-RateLimit-Remaining exceeded"}
		return nil, e
	}
	err = json.NewDecoder(resp.Body).Decode(&pld)
	if err != nil {
		e = ErrStr{Error: fmt.Sprintf("json.NewDecoder(response.Body).Decode(&pld)=%v\n", err)}
		return nil, e
	}
	return pld, ErrStr{Error: ""}
}

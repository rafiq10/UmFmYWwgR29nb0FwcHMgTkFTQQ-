package handlers

import (
	"context"
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

func (s InMemoryPicturesUrlStore) GetPictureUrlByDate(ctx context.Context, from, to string) (pics []string, err error) {
	const dateLayout = "2006-01-02"

	var dates []string

	l := sync.Mutex{}

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

	// check reasonable date range
	for i := 0; i < days+1; i++ {
		dt := startDate.AddDate(0, 0, i)
		dates = append(dates, dt.Format(dateLayout))
	}

	input := make(chan string)
	errch := make(chan error)
	done := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(1)

	loader := func() error {
		defer wg.Done()
		for {
			select {
			case date := <-input:
				pic, err := GetPicturesFromApod(date)
				if err != nil {
					errch <- err
					break
				}
				l.Lock()
				pics = append(pics, pic.Url)
				l.Unlock()
			case <-done:
				return nil
			case <-ctx.Done():
				return nil
			}
		}
	}

	go func() {
		for err := range errch {
			fmt.Printf("Err: %v", err)
			return
		}
	}()
	MAX_CONCURRENT := 5
	for i := 0; i < MAX_CONCURRENT; i++ {
		wg.Add(1)
		go loader()
	}
	for i := 0; i < len(dates); i++ {
		select {
		case <-ctx.Done():
			return nil, nil
		default:
			break
		}
		input <- dates[i]
	}
	close(done)
	wg.Wait()
	close(errch)

	// All data in picsMap
	// Errors received
	// return errors

	return pics, nil
	// for i := 0; i < len(dates); i++ {
	// 	url, ok := picsMap.Load(dates[i])
	// if ok {
	// 	return nil, fmt.Errorf("picsMap.Load(dates[%d])=false", i)
	// }
	// err := time.Parse(dateLayout, dates[i])
	// if err != nil {
	// 	return nil, fmt.Errorf("time.Parse('2006-01-02',%s)=%v", dates[i], err)
	// }
	// ur f            in            t(url)
	// res       apes                fmt.Sprint(url))
	// }
	// l
}
func GetPicturesFromApod(date string) (pld *ApodPayload, e error) {
	var client http.Client

	apiKey := en.GetEnvVar("API_KEY", "DEMO_KEY")

	fullURL := fmt.Sprintf("%s&api_key=%s&date=%s", baseExternalAPI, apiKey, date)
	fmt.Println(fullURL)
	resp, err := client.Get(fullURL)
	if err != nil {
		e = fmt.Errorf("client.Get(%s)=%s\n", fullURL, err.Error())
		return nil, e
	}

	// defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		e = fmt.Errorf("resp.StatusCode=" + strconv.Itoa(resp.StatusCode) + " for URL: " + fullURL)
		return nil, e
	}
	remainingLimit, err := strconv.Atoi(resp.Header.Get("X-RateLimit-Remaining"))
	if err != nil {
		e = fmt.Errorf("Cannot get X-RateLimit-Remaining header: " + err.Error())
		return nil, e
	}
	if remainingLimit < 1 {
		e = fmt.Errorf("X-RateLimit-Remaining exceeded")
		return nil, e
	}
	err = json.NewDecoder(resp.Body).Decode(&pld)
	if err != nil {
		e = fmt.Errorf("json.NewDecoder(response.Body).Decode(&pld)=%v\n", err)
		return nil, e
	}
	return pld, nil
}

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type StubPicturesUrlStore struct {
	urls map[time.Time]string
}

func (s *StubPicturesUrlStore) GetPictureUrlByDate(ctx context.Context, from, to string) ([]string, error) {
	const layout = "2006-01-02"
	var result []string
	start, err := time.Parse(layout, from)
	if err != nil {
		return nil, fmt.Errorf("time.Parse('2022-07-31',%s)=%v", from, err)
	}
	end, err := time.Parse(layout, to)
	if err != nil {
		return nil, fmt.Errorf("time.Parse('2022-07-31',%s)=%v", to, err)
	}
	days := int(end.Sub(start).Hours()/24) + 1

	for i := 0; i < days; i++ {
		result = append(result, s.urls[start.Add(time.Hour*time.Duration(i*24))])
	}

	return result, nil
}

type testDataModel struct {
	SubTestName string
	From        string
	To          string
	Status      int
	Want        Response
	e           ErrStr
}

func TestGETPictures(t *testing.T) {
	store := StubPicturesUrlStore{
		map[time.Time]string{
			time.Date(2022, 6, 29, 0, 0, 0, 0, time.UTC): "https://apod.nasa.gov/apod/image/2206/SolSysPortrait_Trigo_1080_annotated.jpg",
			time.Date(2022, 7, 1, 0, 0, 0, 0, time.UTC):  "https://apod.nasa.gov/apod/image/2207/Star-trail-of-planets1067.jpg",
		},
	}

	testData := []testDataModel{
		{
			"returns a 20220701 picture",
			"2022-07-01",
			"2022-07-01",
			http.StatusOK,
			Response{URLS: []string{"https://apod.nasa.gov/apod/image/2207/Star-trail-of-planets1067.jpg"}},
			ErrStr{},
		},
		{
			"returns a 20220629 picture",
			"2022-06-29",
			"2022-06-29",
			http.StatusOK,
			Response{URLS: []string{"https://apod.nasa.gov/apod/image/2206/SolSysPortrait_Trigo_1080_annotated.jpg"}},
			ErrStr{},
		},
		{
			"returns pictures from 20220629 to 20220701",
			"2022-06-29",
			"2022-07-01",
			http.StatusOK,
			Response{URLS: []string{"https://apod.nasa.gov/apod/image/2206/SolSysPortrait_Trigo_1080_annotated.jpg", "", "https://apod.nasa.gov/apod/image/2207/Star-trail-of-planets1067.jpg"}},
			ErrStr{},
		},
	}

	server := &PictureUrlServer{&store}

	for _, tt := range testData {
		t.Run(tt.SubTestName, func(t *testing.T) {
			request := newGetPictureRequest(tt.From, tt.To)
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			var got Response
			err := json.NewDecoder(response.Body).Decode(&got)
			if err != nil {
				t.Errorf("Unable to parse response from server %q into slice of URLs, '%v'", response.Body, err)
			}

			assertStatus(t, response.Code, tt.Status)
			assertResponseBody(t, fmt.Sprint(got), fmt.Sprint(tt.Want))
		})
	}

	testData = []testDataModel{
		{
			"different picture than expected",
			"2022-07-01",
			"2022-07-01",
			http.StatusOK,
			Response{URLS: []string{"https://apod.nasa.gov/apod/image/2206/SolSysPortrait_Trigo_1080_annotated.jpg"}},
			ErrStr{},
		},
	}

	for _, tt := range testData {
		t.Run(tt.SubTestName, func(t *testing.T) {
			request := newGetPictureRequest(tt.From, tt.To)
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			var got Response
			err := json.NewDecoder(response.Body).Decode(&got)
			if err != nil {
				t.Errorf("Unable to parse response from server %q into slice of URLs, '%v'", response.Body, err)
			}
			assertStatus(t, response.Code, tt.Status)
			assertResponseBodyDifferent(t, fmt.Sprint(got), fmt.Sprint(tt.Want))
		})
	}
	testData = []testDataModel{
		{
			"assure that returns error for the 'from' date",
			"2022",
			"2022-07-01",
			http.StatusBadRequest,
			Response{},
			ErrStr{Error: "'from' argument format not correct"},
		},
		{
			"assure that returns error for the 'from' date",
			"Rafal",
			"2022-07-01",
			http.StatusBadRequest,
			Response{},
			ErrStr{Error: "'from' argument format not correct"},
		},
		{
			"assure that returns error for the 'to' date",
			"2022-06-29",
			"2022",
			http.StatusBadRequest,
			Response{},
			ErrStr{Error: "'to' argument format not correct"},
		},
		{
			"assure that returns specific error for the 'to' date",
			"2022-06-29",
			"",
			http.StatusBadRequest,
			Response{},
			ErrStr{Error: "'to' has to be provided"},
		},
		{
			"assure that returns error when 'to' is earlier than 'from'",
			"2022-07-01",
			"2022-06-29",
			http.StatusBadRequest,
			Response{},
			ErrStr{Error: "'to' cannot be earlier than 'from'"},
		},
	}
	for _, tt := range testData {
		t.Run(tt.SubTestName, func(t *testing.T) {
			request := newGetPictureRequest(tt.From, tt.To)
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			var got Response
			rspTxt := response.Body.String()
			t.Log(rspTxt)
			err := json.NewDecoder(response.Body).Decode(&got)
			if err != nil {
				t.Errorf("Unable to parse response from server %q into slice of URLs, '%v'", response.Body, err)
			}
			assertStatus(t, response.Code, tt.Status)
			assertErrorMsg(t, rspTxt, tt.e)
		})
	}

	tt := testDataModel{
		"check for error if 'from' empty",
		"",
		"2022-07-01",
		http.StatusBadRequest,
		Response{},
		ErrStr{Error: "'from' has to be provided"},
	}
	t.Run(tt.SubTestName, func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/pictures?to=%s", tt.To), nil)
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		var got Response
		rspTxt := response.Body.String()
		t.Log(rspTxt)
		err := json.NewDecoder(response.Body).Decode(&got)
		if err != nil {
			t.Errorf("Unable to parse response from server %q into slice of URLs, '%v'", response.Body, err)
		}
		assertStatus(t, response.Code, tt.Status)
		assertErrorMsg(t, rspTxt, tt.e)
	})
	tt = testDataModel{
		"check for error if 'from' empty",
		"2022-07-01",
		"",
		http.StatusBadRequest,
		Response{},
		ErrStr{Error: "'to' has to be provided"},
	}
	t.Run(tt.SubTestName, func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/pictures?from=%s", tt.From), nil)
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		var got Response
		rspTxt := response.Body.String()
		t.Log(rspTxt)
		err := json.NewDecoder(response.Body).Decode(&got)
		if err != nil {
			t.Errorf("Unable to parse response from server %q into slice of URLs, '%v'", response.Body, err)
		}
		assertStatus(t, response.Code, tt.Status)
		assertErrorMsg(t, rspTxt, tt.e)
	})

}

func newGetPictureRequest(from, to string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/pictures?from=%s&to=%s", from, to), nil)
	return req
}

func assertStatus(t testing.TB, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("did not get correct status, got %d, want %d", got, want)
	}
}

func assertResponseBody(t testing.TB, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Errorf("response body is wrong, got %q want %q", got, want)
	}
}

func assertResponseBodyDifferent(t testing.TB, got, want interface{}) {
	t.Helper()
	if got == want {
		t.Errorf("response body should be different, got %q want %q", got, want)
	}
}
func assertErrorMsg(t testing.TB, bodyStr string, want ErrStr) {
	t.Helper()
	wantStr := `{"error":"` + want.Error
	if wantStr != bodyStr[:len(wantStr)] {
		t.Errorf("Expected error msg: %s, got %s", want.Error, bodyStr)
	}
}

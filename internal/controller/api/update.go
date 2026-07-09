package api

import (
	"bytes"
	"context"
	"errors"
	"github.com/goccy/go-json"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type ERRRespNotOk struct {
	error      error
	StatusCode int
	RespBody   []byte
}

func (e ERRRespNotOk) Error() string {
	return e.error.Error()
}

func UpdateHubStatus(ctx context.Context, updates []UpdateMessage) ([]ControlMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	controllerAPIUrl := ctx.Value("controllerAPIUrl").(*url.URL)
	hubId := ctx.Value("hubId").(string)

	// build URL
	u := controllerAPIUrl.JoinPath("/hub", hubId, "/update")

	updateRequest := struct {
		Updates []UpdateMessage `json:"updates"`
	}{updates}

	body, err := json.Marshal(updateRequest)
	if err != nil {
		return nil, err
	}

	// build request
	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// do request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ERRRespNotOk{
			error:      errors.New("non-200 http response"),
			StatusCode: resp.StatusCode,
			RespBody:   b,
		}
	}

	out := new(struct {
		ControlMessages []ControlMessage `json:"control_messages"`
	})
	err = json.NewDecoder(bytes.NewReader(b)).Decode(out)
	if err != nil {
		return nil, err
	}

	return out.ControlMessages, nil
}

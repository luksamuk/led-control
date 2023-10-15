package wsclient

import (
	"log"
	"fmt"
	"errors"
	"net/http"
	"encoding/json"
)

type Model struct {
	Blinking bool     `json:"blinking,omitempty"`
	Program  int      `json:"program,omitempty"`
	Dim      float64  `json:"dim,omitempty"`
}

const (
	BASEURL = "http://192.168.3.21/led"
)

func parseBody(res *http.Response, err error) (Model, error) {
	var model Model
	if err != nil {
		return model, err
	}

	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return model, errors.New(fmt.Sprintf("HTTP Error: %d", res.StatusCode))
	}

	err = decoder.Decode(&model)
	return model, nil
}

func GetStatus() (Model, error) {
	log.Printf("GET %s", BASEURL)
	res, err := http.Get(BASEURL)
	return parseBody(res, err)
}

func SetAtivo(value bool) (Model, error) {
	var rota string = "on"
	if !value {
		rota = "off"
	}
	url := fmt.Sprintf("%s/%s", BASEURL, rota)

	log.Printf("POST %s", url)
	res, err := http.Post(url, "", nil)
	return parseBody(res, err)
}

func SetDimmer(value float64) (Model, error) {
	url := fmt.Sprintf("%s/dim/%f", BASEURL, value)
	log.Printf("POST %s", url)
	res, err := http.Post(url, "", nil)
	return parseBody(res, err)
}

func ChangeProgram() (Model, error) {
	url := fmt.Sprintf("%s/change", BASEURL)
	log.Printf("POST %s", url)
	res, err := http.Post(url, "", nil)
	return parseBody(res, err)
}

func GetProgramName(program int) string {
	switch program {
	case 0:
		return "Natal"
	case 1:
		return "Rastro"
	case 2:
		return "Lâmpada"
	default:
		return "Desconhecido"
	}
}

package wsclient

import (
	"log"
	"fmt"
	"errors"
	"net/http"
	"encoding/json"
	"image/color"
	"github.com/crazy3lf/colorconv"
)

type Model struct {
	Blinking bool        `json:"blinking,omitempty"`
	Program  int         `json:"program,omitempty"`
	Dim      float64     `json:"dim,omitempty"`
	Strcolor string      `json:"color,omitempty"`
	Color    color.Color
}

const (
	BASEURL = "http://192.168.3.21/led"
)

func ParseHexColor(s string) (c color.NRGBA, err error) {
    c.A = 255
    switch len(s) {
    case 6:
        _, err = fmt.Sscanf(s, "%02x%02x%02x", &c.R, &c.G, &c.B)
    case 3:
        _, err = fmt.Sscanf(s, "%1x%1x%1x", &c.R, &c.G, &c.B)
        // Double the hex digits:
        c.R *= 17
        c.G *= 17
        c.B *= 17
    default:
        err = fmt.Errorf("invalid color length, must be 6 or 3")
    }
	return
}

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
	if err != nil {
		return model, nil
	}

	newcolor, err := ParseHexColor(model.Strcolor)
	if err != nil {
		return Model{}, err
	}

	model.Color = newcolor
	return model, nil
}

func ColorToHex(c color.Color) string {
	return colorconv.ColorToHex(c)[2:]
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

func SetColor(c color.Color) (Model, error) {
	url := fmt.Sprintf("%s/color/%s", BASEURL, ColorToHex(c))
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

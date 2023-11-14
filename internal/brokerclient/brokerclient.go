package brokerclient

import (
	"fmt"
	"log"
	"image/color"
	"strconv"
	"github.com/crazy3lf/colorconv"
	"github.com/denisbrodbeck/machineid"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	brokerurl  = "192.168.3.6:1883"
	brokeruser = "admin"
	brokerpw   = "admin"
	brokersub  = "led/#"
	brokerqos  = 1
)

type Model struct {
	Blinking bool
	Program  int
	Dim      float64
	Color    color.Color
}

var (
	mClient mqtt.Client
	state = Model{
		Blinking: false,
		Program: 2,
		Dim: 1.0,
		Color: color.NRGBA{R: 255, G: 255, B: 255, A: 255},
	}
)

// MQTT Configuration
func Init(appname string) error {
	id, err := machineid.ProtectedID(appname)
	if err != nil {
		return err
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerurl)
	opts.SetClientID(id)
	opts.SetUsername(brokeruser)
	opts.SetPassword(brokerpw)
	opts.SetDefaultPublishHandler(mqttPublishHandler)
	opts.OnConnect = mqttConnectHandler
	opts.OnConnectionLost = mqttConnectionLostHandler
	opts.SetAutoReconnect(true)
	mClient = mqtt.NewClient(opts)

	log.Printf("Connecting to MQTT Broker @ %s...", brokerurl)
	conntoken := mClient.Connect()
	if conntoken.Wait() && conntoken.Error() != nil {
		return conntoken.Error()
	}

	log.Printf("Subscribing to %s (QOS=%d)...", brokersub, brokerqos)
	subtoken := mClient.Subscribe(brokersub, brokerqos, nil)
	subtoken.Wait()

	log.Print("MQTT config finished.")
	return nil
}

func mqttPublishHandler(client mqtt.Client, msg mqtt.Message) {
	payload := string(msg.Payload()[:])
	switch msg.Topic() {
	case "led/active":
		if n, err := strconv.Atoi(payload); err == nil {
			state.Blinking = (n == 1)
		}
	case "led/dim":
		if n, err := strconv.ParseFloat(payload, 64); err == nil {
			state.Dim = n
		}
	case "led/program":
		if n, err := strconv.Atoi(payload); err == nil {
			if (n < 0) || (n > 2) {
				n = 2
			}
			state.Program = n
		}
	case "led/color":
		if c, err := ParseHexColor(payload); err == nil {
			state.Color = c
		}
	default:
		log.Printf("Ignoring topic %s", msg.Topic())
	}
}

func mqttConnectHandler(client mqtt.Client) {
	log.Printf("Connection established to MQTT Broker at %s", brokerurl)
}

func mqttConnectionLostHandler(client mqtt.Client, err error) {
	log.Printf("Connection lost to MQTT Broker: %v", err)
}

func publishMessage(topic string, value string) error {
	log.Printf("pub %s <- %s (qos=%d, retained=true)...", topic, value, brokerqos)
	token := mClient.Publish(topic, brokerqos, true, []byte(value))
	<-token.Done()
	return token.Error()
}

// Getters and setters

func GetStatus() Model {
	return state
}

func SetAtivo(value bool) error {
	v := 0
	if value {
		v = 1
	}
	return publishMessage("led/active", fmt.Sprintf("%d", v))
}

func SetDimmer(value float64) error {
	return publishMessage("led/dim", fmt.Sprintf("%f", value))
}

func SetColor(c color.Color) error {
	col := ColorToHex(c)
	if col == "000000" {
		return nil
	}
	return publishMessage("led/color", col)
}

func SetProgram(program string) error {
	p := GetProgramIndex(program)
	return publishMessage("led/program", fmt.Sprintf("%d", p))
}

// Util functions
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

func ColorToHex(c color.Color) string {
	return colorconv.ColorToHex(c)[2:]
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

func GetProgramIndex(program string) int {
	switch program {
	case "Natal":
		return 0
	case "Rastro":
		return 1
	case "Lâmpada":
		fallthrough
	default:
		return 2
	}
}


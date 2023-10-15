package main

import (
	"log"
	"time"
	"math"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	xtheme "fyne.io/x/fyne/theme"
	xlayout "fyne.io/x/fyne/layout"
	"github.com/lusingander/colorpicker"

	"com.luksamuk.ledcontrol/wsclient"
)

var (
	a              fyne.App
	gotFirstValues bool
	gstatus        wsclient.Model
	lastDimValue   float64
	lastColorValue color.Color
)

func percentToDim(value float64) float64 {
	if value < 0.0 {
		return 0.02
	} else if value > 100.0 {
		return 1.0
	}
	
	return 0.02 + ((value / 100.0) * (1.0 - 0.02))
}

func dimToPercent(value float64) float64 {
	if value < 0.02 {
		return 0.0
	} else if value > 1.0 {
		return 100.0
	}

	result := (value * 100) / (1.0 - 0.02)
	if result > 100.0 {
		return 100.0
	}
	return result
}

func main() {
	gotFirstValues = false
	a = app.NewWithID("com.luksamuk.ledcontrol")

	a.Settings().SetTheme(xtheme.AdwaitaTheme())

	w := a.NewWindow("Controle de LED")

	// Global function for refreshing the global state
	refreshState := func(m wsclient.Model) {
		gstatus = m
	}

	// Active/inactive checkbox
	chkBlink := widget.NewCheck("Ligado/Desligado", func(value bool) {
		if gotFirstValues {
			go func() {
				res, err := wsclient.SetAtivo(value)
				if err != nil {
					log.Printf("Erro: %v", err)
					return
				}
				refreshState(res)
			}()
		}
	})

	// Dim slider
	sldDim := widget.NewSlider(0.0, 100.0)
	sldDim.OnChanged = func(value float64) {
		actualValue := percentToDim(value)
		if gotFirstValues {
			lastDimValue = actualValue
		}
	}

	// Dim service for asynchonously changing dim.
	// This is made this way since the Raspberry Pi Pico W can't take a big
	// load of requests. So instead of sending a request every time the slider
	// changes, we perform requests with respect to the amount that the slider
	// changed since the last time we sent a dim change request.
	go func() {
		for {
			diff := math.Abs(lastDimValue - gstatus.Dim)
			if diff < 0.02 {
				time.Sleep(200 * time.Millisecond)
				continue
			}

			delta := math.Max(0.02, diff)
			if lastDimValue < gstatus.Dim {
				delta *= -1.0
			}

			res, err := wsclient.SetDimmer(gstatus.Dim + delta)
			if err != nil {
				log.Printf("Erro: %v", err)
				continue
			}
			refreshState(res)
		}
	}()

	cmbProgram := widget.NewSelect(
		[]string{"Natal", "Rastro", "Lâmpada"},
		func(option string) {
			if gotFirstValues {
				res, err := wsclient.SetProgram(option)
				if err != nil {
					log.Printf("Erro: %v", err)
					return
				}
				refreshState(res)
			}
		})

	// Button for cycling the current program
	btnChangeProgram := widget.NewButtonWithIcon(
		"",
		theme.MediaReplayIcon(),
		func() {
			res, err := wsclient.ChangeProgram()
			if err != nil {
				log.Printf("Erro: %v", err)
				return
			}
			refreshState(res)
		})

	// Color picker for setting up the lamp color
	picker := colorpicker.New(200, colorpicker.StyleValue)
	picker.SetOnChanged(func(c color.Color) {
		if gotFirstValues {
			lastColorValue = c
		}
	})
	
	// Behaviour for refreshing everything and obtaining the global status
	// again from the remote device
	refreshAll := func() {
		gotFirstValues = false
		status, err := wsclient.GetStatus()
		if err == nil {
			refreshState(status)
			chkBlink.SetChecked(status.Blinking)
			sldDim.SetValue(dimToPercent(status.Dim))
			picker.SetColor(status.Color)
			cmbProgram.SetSelected(wsclient.GetProgramName(status.Program))
			lastDimValue = status.Dim
			lastColorValue = status.Color
			gotFirstValues = true
		} else {
			log.Printf("Error: %v", err)
		}
	}

	// Refresh global state button
	btnRefreshAll := widget.NewButtonWithIcon(
		"Atualizar Tudo",
		theme.ViewRefreshIcon(),
		refreshAll)

	btnResetColor := widget.NewButtonWithIcon(
		"Resetar Cor",
		theme.ContentUndoIcon(),
		func() {
			white, _ := wsclient.ParseHexColor("ffffff")
			picker.SetColor(white)
		})

	// This is a goroutine that works just like the one for the dimmer. But
	// this time, we are responsibly changing colors in a way that does not
	// do an enormous amount of load to the remote device
	go func() {
		for {
			if gotFirstValues {
				time.Sleep(200 * time.Millisecond)
				if wsclient.ColorToHex(lastColorValue) != wsclient.ColorToHex(gstatus.Color) {
					res, err := wsclient.SetColor(lastColorValue)
					if err != nil {
						log.Printf("Erro: %v", err)
						continue
					}
					refreshState(res)
				}
			}
		}
	}()

	// Some labels
	lblIntensidade := widget.NewLabel("Intensidade:")
	lblProgramacao := widget.NewLabel("Programa:")

	// Responsive layout within the application
	l := xlayout.NewResponsiveLayout(
		xlayout.Responsive(chkBlink, .35, .35),
		xlayout.Responsive(widget.NewLabel(""), .65, .65),

		xlayout.Responsive(lblProgramacao, .35, .35),
		xlayout.Responsive(cmbProgram, .55, .55),
		xlayout.Responsive(btnChangeProgram, .1, .1),

		xlayout.Responsive(lblIntensidade, 1, .35),
		xlayout.Responsive(sldDim, 1, .65),

		xlayout.Responsive(widget.NewLabel(""), 1, 1),

		xlayout.Responsive(picker, 1, 1),
		
		xlayout.Responsive(widget.NewLabel(""), 1, 1),
		xlayout.Responsive(btnRefreshAll, .5, .5),
		xlayout.Responsive(btnResetColor, .5, .5),
	)

	// Application header and footer
	title := widget.NewLabel("Controle de LED")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter
	
	header := xlayout.NewResponsiveLayout(
		xlayout.Responsive(title, 1, 1),
	)

	footer := xlayout.NewResponsiveLayout(
		xlayout.Responsive(widget.NewLabel(""), 1, 1),
	)

	content := fyne.NewContainerWithLayout(
		layout.NewBorderLayout(header, footer, nil, nil),
		header,
		footer,
		l,
	)

	w.SetContent(content)


	// Perform first update on application launch
	refreshAll()

	w.Resize(fyne.NewSize(360, 640))
	w.CenterOnScreen()
	w.ShowAndRun()
}


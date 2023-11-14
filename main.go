package main

import (
	"fmt"
	"log"
	"time"
	"math"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"fyne.io/fyne/v2/container"
	xtheme "fyne.io/x/fyne/theme"
	xlayout "fyne.io/x/fyne/layout"
	"github.com/lusingander/colorpicker"

	client "com.luksamuk.ledcontrol/brokerclient"
)

const APPNAME = "com.luksamuk.ledcontrol"

var (
	a              fyne.App
	gotFirstValues bool
	gstatus        client.Model
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
	a = app.NewWithID(APPNAME)

	a.Settings().SetTheme(xtheme.AdwaitaTheme())
	
	w := a.NewWindow("Controle de LED")

	// Log scroll pane
	logwindow := widget.NewTextGrid()
	scrollpane := container.NewScroll(logwindow)

	// Logging function
	logappend := func(text string) {
		log.Print(text)
		logwindow.SetText(logwindow.Text() + "\n" + text)
		scrollpane.ScrollToBottom()
	}

	logerr := func(err error) {
		if err != nil {
			logappend(fmt.Sprintf("[!] %v", err))
		}
	}

	// Global function for refreshing the global state
	refreshState := func() {
		gstatus = client.GetStatus()
	}

	// Active/inactive checkbox
	chkBlink := widget.NewCheck("Ligado/Desligado", func(value bool) {
		if gotFirstValues {
			go func() {
				logerr(client.SetAtivo(value))
				refreshState()
			}()
		}
	})

	// Dim slider
	sldDim := widget.NewSlider(0.0, 100.0)
	sldDim.OnChanged = func(value float64) {
		actualValue := percentToDim(value)
		if gotFirstValues {
			lastDimValue = actualValue
			refreshState()
		}
	}

	// Dim service for asynchonously changing dim.
	// This is made this way since the Raspberry Pi Pico W can't take a big
	// load of changes, and we don't wanna fill the broker with messages.
	// So instead of sending a message every time the slide changes, we
	// send them with respect to the amount that the slider
	// changed since the last time we sent a dim change.
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

			logerr(client.SetDimmer(gstatus.Dim + delta))
			refreshState()
		}
	}()

	cmbProgram := widget.NewSelect(
		[]string{"Natal", "Rastro", "Lâmpada"},
		func(option string) {
			if gotFirstValues {
				logerr(client.SetProgram(option))
				refreshState()
			}
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
		refreshState()
		chkBlink.SetChecked(gstatus.Blinking)
		sldDim.SetValue(dimToPercent(gstatus.Dim))
		picker.SetColor(gstatus.Color)
		cmbProgram.SetSelected(client.GetProgramName(gstatus.Program))
		lastDimValue = gstatus.Dim
		lastColorValue = gstatus.Color
		gotFirstValues = true
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
			white, _ := client.ParseHexColor("ffffff")
			picker.SetColor(white)
		})

	// This is a goroutine that works just like the one for the dimmer. But
	// this time, we are responsibly changing colors in a way that does not
	// do an enormous amount of load to the remote device
	go func() {
		for {
			if gotFirstValues {
				time.Sleep(200 * time.Millisecond)
				if client.ColorToHex(lastColorValue) != client.ColorToHex(gstatus.Color) {
					logerr(client.SetColor(lastColorValue))
					refreshState()
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
		xlayout.Responsive(cmbProgram, .65, .65),

		xlayout.Responsive(lblIntensidade, 1, 1),
		xlayout.Responsive(sldDim, 1, 1),

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
		//xlayout.Responsive(widget.NewLabel(""), 1, 1),
		xlayout.Responsive(scrollpane, 1, 1),
	)

	content := fyne.NewContainerWithLayout(
		layout.NewBorderLayout(header, footer, nil, nil),
		header,
		footer,
		l,
	)

	w.SetContent(content)


	// Perform first update on application launch
	logerr(client.Init(APPNAME))
	time.Sleep(500 * time.Millisecond)
	refreshAll()

	w.Resize(fyne.NewSize(360, 640))
	w.CenterOnScreen()
	w.ShowAndRun()
}


package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type APIRequest struct {
	HTTPMethod   string
	Path         string
	ParamsJSON   string
	UserAgent    string
	AuthUsername string
	AuthPassword string
	Referrer     string
	BurstModeRPS uint64
}

var apiRequest = &APIRequest{}

type AppSettings struct {
	SetDarkTheme        bool    `json:"dark_theme_set"`
	WindowSizeX         float32 `json:"window_size_x"`
	WindowSizeY         float32 `json:"window_size_Y"`
	HideRequestHeaders  bool    `json:"hide_request_headers"`
	HideResponseHeaders bool    `json:"hide_response_headers"`
}

var appSettings = &AppSettings{}
var settingsFileName = "settings.json"

var httpMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
var httpClient = &http.Client{}

var responsesLabel = widget.NewLabel("")
var responseProgressBar widget.ProgressBarInfinite

var darkModeCheck, hideRequestHeadersCheck, hideResponseHeadersCheck *widget.Check

const (
	defaultWindowSizeX = 1250
	defaultWindowSizeY = 600
)

func main() {
	a := app.New()
	w := a.NewWindow("Gui4REST")
	w.CenterOnScreen()

	initAppSettings(a, w) //read settings from file to appSettingsStruct then apply settings

	vscroll := container.NewVScroll(populateVBox2()) //wrap vbox in a vscroll first to make it scrollable
	vscroll.Direction = container.ScrollVerticalOnly

	mainGridColumns := container.NewGridWithColumns(3, populateVBox1(), vscroll, populateVBox3(a))
	mainGridRow := container.NewGridWithRows(1, mainGridColumns) //only way to extend the VBox to the whole height is to use Grids

	w.SetContent(container.NewVBox(
		mainGridRow,
	))

	w.ShowAndRun()
}

func populateVBox1() *fyne.Container {
	httpMethodSelect := widget.NewSelect(httpMethods, func(value string) {
		apiRequest.HTTPMethod = value
	})
	httpMethodSelect.SetSelectedIndex(0)

	pathEntry := widget.NewEntry()
	pathEntry.SetPlaceHolder("https://api.example.com:8080/users/:param")
	pathEntry.OnChanged = func(value string) {
		apiRequest.Path = value
	}

	paramsTextarea := widget.NewMultiLineEntry()
	paramsTextarea.SetPlaceHolder(`{"key1":"value1","key2":"value2",...}`)
	paramsTextarea.OnChanged = func(value string) {
		apiRequest.ParamsJSON = value
	}

	burstRPSEntry := widget.NewEntry() //RPS => Requests Per Second
	burstRPSEntry.SetPlaceHolder("requests per second")
	burstRPSEntry.OnChanged = func(rpsStr string) {
		if converted, err := strconv.ParseUint(rpsStr, 10, 64); err == nil {
			apiRequest.BurstModeRPS = converted
		}
	}
	burstRPSEntry.Hide() //proper implementation in the pipeline. Patience :-)

	burstModeCheck := widget.NewCheck("Burst Mode (test rate-limiting & loading)", func(isChecked bool) {
		if isChecked {
			burstRPSEntry.Show()
		} else {
			burstRPSEntry.Hide()
			apiRequest.BurstModeRPS = 0 // reset in case it was initially set
		}
	})

	sendButton := widget.NewButton("Send Request", func() {
		sendHTTPRequest()
	})

	saveButton := widget.NewButton("Save Request", func() {
		saveAPIRequest()
	})
	saveButton.Disable() //proper implementation in the pipeline. Patience :-)

	vbox1 := container.NewVBox(
		titleCanvasText(canvas.NewText("REQUESTS", nil)),
		widget.NewLabel("\r\nEndpoint Path"),
		pathEntry,
		widget.NewLabel("\r\nHTTP Method"),
		httpMethodSelect,
		widget.NewLabel("\r\nParameters (JSON)"),
		paramsTextarea,
		burstModeCheck,
		burstRPSEntry,
		container.NewGridWithColumns(2, sendButton, saveButton),
	)

	return vbox1
}

func populateVBox2() *fyne.Container {
	responseProgressBar = *widget.NewProgressBarInfinite()

	responsesLabel.Wrapping = fyne.TextWrapWord // set word wrap to avoid out-stretching the width

	vbox2 := container.NewVBox(
		titleCanvasText(canvas.NewText("RESPONSES", nil)),
		&responseProgressBar,
		responsesLabel,
	)
	responseProgressBar.Hide() //only show it once request is sent

	return vbox2
}

func populateVBox3(a fyne.App) *fyne.Container {
	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("Username")
	usernameEntry.OnChanged = func(value string) {
		apiRequest.AuthUsername = value
		responsesLabel.SetText("Config changed. Re-send request")
	}

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Password")
	passwordEntry.OnChanged = func(value string) {
		apiRequest.AuthPassword = value
		responsesLabel.SetText("Config changed. Resend request")
	}

	uaTextarea := widget.NewMultiLineEntry()
	uaTextarea.SetPlaceHolder("User Agent")
	uaTextarea.OnChanged = func(value string) {
		apiRequest.UserAgent = value
		responsesLabel.SetText("Config changed. Resend request")
	}

	referrerTextarea := widget.NewMultiLineEntry()
	referrerTextarea.SetPlaceHolder("Referrer")
	referrerTextarea.OnChanged = func(value string) {
		apiRequest.Referrer = value
		responsesLabel.SetText("Config changed. Resend request")
	}

	hideRequestHeadersCheck = widget.NewCheck("Hide Req Headers", func(checked bool) {
		if checked {
			appSettings.HideRequestHeaders = true
			saveAppSettings()
			responsesLabel.SetText("Resend request to effect change")
		} else {
			appSettings.HideRequestHeaders = false
			saveAppSettings()
			responsesLabel.SetText("Resend request to effect change")
		}
	})

	hideResponseHeadersCheck = widget.NewCheck("Hide Resp Headers", func(checked bool) {
		if checked {
			appSettings.HideResponseHeaders = true
			saveAppSettings()
			responsesLabel.SetText("Resend request to effect change")
		} else {
			appSettings.HideResponseHeaders = false
			saveAppSettings()
			responsesLabel.SetText("Resend request to effect change")
		}
	})

	darkModeCheck = widget.NewCheck("Dark Mode", func(checked bool) {
		if checked {
			a.Settings().SetTheme(theme.DarkTheme())
			appSettings.SetDarkTheme = true
			saveAppSettings()
		} else {
			a.Settings().SetTheme(theme.LightTheme())
			appSettings.SetDarkTheme = false
			saveAppSettings()
		}
	})
	initialCheckUncheck(a) //check/uncheck on app startup. Doing it later results in segfaulting

	creditsLicensesButton := widget.NewButton("Credits & Licenses", func() {
		creditsLicenses(a)
	})

	vbox3 := container.NewVBox(
		titleCanvasText(canvas.NewText("CONFIGURATION", nil)),
		widget.NewLabel("\r\nAuthentication"),
		container.NewGridWithColumns(2, usernameEntry, passwordEntry),
		widget.NewLabel("\r\nUser Agent"),
		uaTextarea,
		widget.NewLabel("\r\nReferrer"),
		referrerTextarea,
		container.NewGridWithColumns(2, hideRequestHeadersCheck, hideResponseHeadersCheck),
		darkModeCheck,
		creditsLicensesButton,
	)

	return vbox3
}

func initialCheckUncheck(a fyne.App) {
	//check them appropriately on app startup
	if appSettings.HideRequestHeaders {
		hideRequestHeadersCheck.SetChecked(true)
	}
	if appSettings.HideResponseHeaders {
		hideResponseHeadersCheck.SetChecked(true)
	}

	// apply theme on app startup
	if appSettings.SetDarkTheme {
		a.Settings().SetTheme(theme.DarkTheme())
		darkModeCheck.SetChecked(true)
	} else {
		a.Settings().SetTheme(theme.LightTheme())
	}
}

func titleCanvasText(ct *canvas.Text) *canvas.Text {
	ct.TextSize = 16
	ct.TextStyle = fyne.TextStyle{Bold: true}
	ct.Alignment = fyne.TextAlign(fyne.TextAlignCenter)

	return ct
}

func initAppSettings(a fyne.App, w fyne.Window) {
	var contentBytes []byte

	// only create new settings file is !exists
	if _, err := os.Stat(settingsFileName); err != nil {
		contentBytes = createSettingsFile()
	} else if contentBytes, err = ioutil.ReadFile(settingsFileName); err != nil { //read the file
		responsesLabel.SetText("ERROR: \r\n" + err.Error())
		return
	}

	// umarshal into appSettings struct
	if err := json.Unmarshal(contentBytes, &appSettings); err != nil {
		responsesLabel.SetText("ERROR: \r\n" + err.Error())
		return
	}

	// apply window size
	w.Resize(fyne.NewSize(appSettings.WindowSizeX, appSettings.WindowSizeY))
}

func createSettingsFile() []byte {
	// to avoid zero sized window, set default window size
	appSettings.WindowSizeX = defaultWindowSizeX
	appSettings.WindowSizeY = defaultWindowSizeY

	// marshal struct to JSON
	contentBytes, err := json.Marshal(appSettings)
	if err != nil {
		log.Fatal(err)
	}

	// write json to settings file
	if err := ioutil.WriteFile(settingsFileName, contentBytes, 0644); err != nil {
		log.Fatal(err)
	}

	return contentBytes
}

func saveAppSettings() {
	b, err := json.Marshal(appSettings) // create the JSON to be written
	if err != nil {
		log.Fatal(err)
	}

	// write the JSON to file. Existing content will be overwritten
	if err2 := ioutil.WriteFile(settingsFileName, b, 0644); err2 != nil {
		log.Fatal(err2)
	}
}

func sendHTTPRequest() {
	responseProgressBar.Show()
	responsesLabel.SetText("")

	var responseToPrint string

	timeStart := time.Now().UnixNano() // round-trip timer
	req, err := http.NewRequest(apiRequest.HTTPMethod, apiRequest.Path, bytes.NewBuffer([]byte(apiRequest.ParamsJSON)))
	if err != nil {
		responsesLabel.SetText("ERROR: \r\n" + err.Error())
		responseProgressBar.Hide()
		return
	}

	// set request headers
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", apiRequest.UserAgent)
	req.Header.Set("Referer", apiRequest.Referrer)
	req.SetBasicAuth(apiRequest.AuthUsername, apiRequest.AuthPassword)

	resp, err := httpClient.Do(req)
	if err != nil {
		responsesLabel.SetText("ERROR: \r\n" + err.Error())
		responseProgressBar.Hide()
		return
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		responsesLabel.SetText("ERROR: \r\n" + err.Error())
		responseProgressBar.Hide()
		return
	}

	roundtripMillis := (time.Now().UnixNano() - timeStart) / 1000000
	responseToPrint = formatServerResponse(req, resp, string(respBody), roundtripMillis)

	responsesLabel.SetText(responseToPrint)
	responseProgressBar.Hide()
}

func formatServerResponse(req *http.Request, resp *http.Response, bodyStr string, roundtripMillis int64) string {
	requestHeadersStr := ""
	for key, value := range req.Header {
		requestHeadersStr += key + " : " + strings.Join(value, ", ") + "\r\n"
	}
	responseHeadersStr := ""
	for key, value := range resp.Header {
		responseHeadersStr += key + " : " + strings.Join(value, ", ") + "\r\n"
	}

	responseToPrint := "Round-trip ime: " + fmt.Sprintf("%v", roundtripMillis) + "ms \r\n" +
		"HTTP status: " + resp.Status + "\r\n\r\n"

	if !appSettings.HideRequestHeaders {
		responseToPrint += "REQUEST HEADERS: \r\n" + requestHeadersStr + "\r\n"
	}
	if !appSettings.HideResponseHeaders {
		responseToPrint += "RESPONSE HEADERS: \r\n" + responseHeadersStr + "\r\n"
	}

	responseToPrint += "RESPONSE BODY: \r\n" + bodyStr

	return responseToPrint
}

func creditsLicenses(a fyne.App) {
	creditsLabel := widget.NewLabel("CREDITS\r\n" +
		"Created By Martin Ombiro on a fine Monday afternoon.\r\n" +
		"Happy REST API grokking!\r\n\r\n\r\n" +
		"CONTACT\r\n" +
		"martinno@tutanota.com")

	licensesLabel := widget.NewLabel("\r\nLICENSES\r\n" +
		"This application and its source code is governed by the\r\n" +
		"GNU General Public License (GPL) 3.0 license")

	w := a.NewWindow("Licenses & Credits")
	w.SetContent(
		container.NewVBox(
			creditsLabel,
			licensesLabel,
		),
	)
	w.Resize(fyne.NewSize(350, 350))
	w.CenterOnScreen()
	w.Show()
}

func saveAPIRequest() {
	//TODO
}

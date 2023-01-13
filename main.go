package main

import (
	"encoding/json"
	"fmt"
	"github.com/brokiem/auto-hoyolab-checkin/icon"
	"github.com/getlantern/systray"
	"github.com/go-co-op/gocron"
	"github.com/gonutz/w32/v2"
	"github.com/zellyn/kooky"
	_ "github.com/zellyn/kooky/browser/chrome"
	_ "github.com/zellyn/kooky/browser/firefox"
	_ "github.com/zellyn/kooky/browser/opera"
	_ "github.com/zellyn/kooky/browser/safari"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"time"
)

var actId = "nil"
var autoHide = false

func main() {
	s := gocron.NewScheduler(time.UTC)
	s.Every(12).Hours().Do(RunProgram)
	s.StartAsync()

	fmt.Println(" \nAutomatic Hoyolab Check-in (https://github.com/brokiem/auto-hoyolab-checkin) \n\n[DO NOT CLOSE THIS WINDOW]\nTo minimize or hide this window, \nclick the icon in the SYSTEM TRAY then choose \"Hide window\" button\n ")

	systray.Run(onReady, onExit)
}

func hideConsole() {
	console := w32.GetConsoleWindow()
	if console == 0 {
		return // no console attached
	}
	_, consoleProcID := w32.GetWindowThreadProcessId(console)
	if w32.GetCurrentProcessId() == consoleProcID {
		w32.ShowWindowAsync(console, w32.SW_HIDE)
	}
}

func showConsole() {
	console := w32.GetConsoleWindow()
	if console == 0 {
		return // no console attached
	}
	_, consoleProcID := w32.GetWindowThreadProcessId(console)
	if w32.GetCurrentProcessId() == consoleProcID {
		w32.ShowWindowAsync(console, w32.SW_SHOW)
	}
}

func RunProgram() {
	ReadConfiguration()

	if autoHide {
		hideConsole()
	}

	token := kooky.ReadCookies(kooky.Valid, kooky.DomainHasSuffix(`.hoyolab.com`), kooky.Name(`ltoken`))
	ltuid := kooky.ReadCookies(kooky.Valid, kooky.DomainHasSuffix(`.hoyolab.com`), kooky.Name(`ltuid`))

	if len(token) <= 0 {
		fmt.Println("Account TOKEN not found, please login to hoyolab once in Chrome/Firefox/Opera/Safari")
		return
	}
	if len(ltuid) <= 0 {
		fmt.Println("Account LTUID not found, please re-login in your browser")
		return
	}

	isClaimed := GetClaimedStatus(token[0], ltuid[0])

	if !isClaimed {
		ClaimReward(token[0], ltuid[0])
		fmt.Println("You've claimed your reward today!")
	} else {
		fmt.Println("Traveler, you've already checked in today~")
	}
}

type Config struct {
	ActId          string
	AutoHideWindow bool
}

func ReadConfiguration() {
	if _, err := os.Stat("config.json"); err == nil {
	} else {
		configMap := Config{ActId: "e202102251931481", AutoHideWindow: false}
		jsonByte, _ := json.MarshalIndent(configMap, "", " ")

		_ = ioutil.WriteFile("config.json", jsonByte, 0644)
	}

	jsonFile, _ := os.Open("config.json")
	byteValue, _ := ioutil.ReadAll(jsonFile)

	var result map[string]interface{}
	json.Unmarshal(byteValue, &result)

	actId = reflect.ValueOf(result["ActId"]).String()
	autoHide = reflect.ValueOf(result["AutoHideWindow"]).Bool()
}

func GetClaimedStatus(token *kooky.Cookie, ltuid *kooky.Cookie) bool {
	req, _ := http.NewRequest("GET", "https://sg-hk4e-api.hoyolab.com/event/sol/info", nil)

	params := url.Values{}
	params.Add("act_id", actId)
	req.URL.RawQuery = params.Encode()

	req.Header.Add("Accept", "application/json, text/plain, */*")
	req.Header.Add("Accept-Language", "en-US,en;q=0.5")
	req.Header.Add("Origin", "https://act.hoyolab.com")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Referer", "https://act.hoyolab.com/ys/event/signin-sea-v3/index.html?act_id="+actId)
	req.Header.Add("Cache-Control", "'max-age=0")

	req.AddCookie(&http.Cookie{
		Name:   token.Name,
		Value:  token.Value,
		MaxAge: token.MaxAge,
	})
	req.AddCookie(&http.Cookie{
		Name:   ltuid.Name,
		Value:  ltuid.Value,
		MaxAge: ltuid.MaxAge,
	})

	client := &http.Client{}
	resp, _ := client.Do(req)

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyString := string(bodyBytes)

	var result map[string]interface{}
	json.Unmarshal([]byte(bodyString), &result)

	return reflect.ValueOf(result["data"].(map[string]interface{})["is_sign"]).Bool()
}

func ClaimReward(token *kooky.Cookie, ltuid *kooky.Cookie) string {
	m := map[string]string{"act_id": actId}
	read, write := io.Pipe()
	go func() {
		json.NewEncoder(write).Encode(m)
		write.Close()
	}()

	req, _ := http.NewRequest("POST", "https://sg-hk4e-api.hoyolab.com/event/sol/sign", read)

	req.Header.Add("Accept", "application/json, text/plain, */*")
	req.Header.Add("Accept-Language", "en-US,en;q=0.5")
	req.Header.Add("Content-Type", "application/json;charset=utf-8")
	req.Header.Add("Origin", "https://act.hoyolab.com")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Referer", "https://act.hoyolab.com/ys/event/signin-sea-v3/index.html?act_id="+actId)

	req.AddCookie(&http.Cookie{
		Name:   token.Name,
		Value:  token.Value,
		MaxAge: token.MaxAge,
	})
	req.AddCookie(&http.Cookie{
		Name:   ltuid.Name,
		Value:  ltuid.Value,
		MaxAge: ltuid.MaxAge,
	})

	client := &http.Client{}
	resp, _ := client.Do(req)

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyString := string(bodyBytes)

	var result map[string]interface{}
	json.Unmarshal([]byte(bodyString), &result)

	return reflect.ValueOf(result["message"]).String()
}

func onReady() {
	systray.SetTemplateIcon(icon.Data, icon.Data)
	systray.SetTitle("Automatic Hoyolab Check-in")
	systray.SetTooltip("Automatic Hoyolab Check-in")

	bShow := systray.AddMenuItem("Show window", "Show console")
	bHide := systray.AddMenuItem("Hide window", "Hide console")
	systray.AddSeparator()
	bExit := systray.AddMenuItem("Exit", "Exit the whole app")
	go func() {
		for {
			select {
			case <-bHide.ClickedCh:
				hideConsole()
			case <-bShow.ClickedCh:
				showConsole()
			case <-bExit.ClickedCh:
				systray.Quit()
			}
		}
	}()
}

func onExit() {
	fmt.Println("Exiting...")
}

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/signal"
	"time"

	"strconv"
	"strings"

	"net/http"

	"github.com/samuelb-web/railmiles-richpresence/icon"

	"github.com/PuerkitoBio/goquery"
	"github.com/getlantern/systray"
	"github.com/hugolgst/rich-go/client"
	"github.com/robfig/cron"
	"gopkg.in/yaml.v3"
)

type LeagueUser struct {
	Username string
	Position int
	Miles    int
	Chains   int
}

type LeagueData struct {
	Name        string
	CurrentUser LeagueUser
	Users       []LeagueUser
}

type IndividualData struct {
	Miles  int
	Chains int
}

type IndividualJSONData struct {
	Totals struct {
		Distance string `json:"distance"`
	} `json:"totals"`
}

type Config struct {
	ApplicationId string `yaml:"applicationId"`
	Username      string `yaml:"username"`

	RailmilesUrl string `yaml:"url"`

	Mode   string `yaml:"mode"`
	League int    `yaml:"league"`

	Messages struct {
		TopSoloMessage   string `yaml:"topSoloMessage"`
		TopLeagueMessage string `yaml:"topLeagueMessage"`

		BottomSoloMessage   string `yaml:"bottomSoloMessage"`
		BottomLeagueMessage string `yaml:"bottomLeagueMessage"`
	} `yaml:"messages"`

	Buttons struct {
		UserLink struct {
			Visible bool   `yaml:"visible"`
			Message string `yaml:"message"`
		} `yaml:"userLink"`

		LeagueLink struct {
			Visible bool   `yaml:"visible"`
			Message string `yaml:"message"`
		} `yaml:"leagueLink"`
	} `yaml:"buttons"`

	ImageKeys struct {
		Large string `yaml:"largeImage"`
		Small string `yaml:"smallImage"`
	} `yaml:"imageKeys"`
}

func getPositionSuffix(position int) string {
	switch position {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	default:
		return "th"
	}
}

func getLeagueData(config Config) LeagueData {
	leagueData := LeagueData{}

	// Get the Railmiles league page
	response, err := http.Get("https://my.railmiles.me/leagues/" + strconv.Itoa(config.League))
	if err != nil {
		log.Fatalf("Error getting league data: %s\n", err.Error())
		return leagueData
	}

	defer response.Body.Close()

	// Parse the HTML data from the response body to extract the datapoints we want
	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Fatalf("Error parsing league data: %s\n", err.Error())
		return leagueData
	}

	// Pull league name
	leagueData.Name = doc.Find("h1").Nodes[1].FirstChild.Data

	// Pull league user data
	members := doc.Find(".league-member")

	for i := 0; i < members.Length(); i++ {
		element := members.Eq(i)

		username := element.Find(".user").Nodes[0].FirstChild.Data
		distance := element.Find(".distance").Nodes[0].FirstChild.Data

		username = strings.TrimSpace(username)
		distance = strings.TrimSpace(distance)

		distanceStrings := strings.Split(distance, " ")
		miles, _ := strconv.Atoi(strings.TrimSuffix(distanceStrings[0], "mi"))
		chains, _ := strconv.Atoi(strings.TrimSuffix(distanceStrings[1], "ch"))

		leagueUser := LeagueUser{Username: username, Position: (i + 1), Miles: miles, Chains: chains}
		leagueData.Users = append(leagueData.Users, leagueUser)

		if username == config.Username {
			leagueData.CurrentUser = leagueUser
		}
	}

	return leagueData
}

func getIndividualData(config Config) IndividualData {
	individualData := IndividualData{}

	// Get the current year
	year := time.Now().Year()

	// Get the Railmiles journeys for the specified user
	response, err := http.Get(config.RailmilesUrl + "jsearch?year=" + strconv.Itoa(year))
	if err != nil {
		log.Fatalf("Error getting individual data: %s\n", err.Error())
		return individualData
	}

	if response.Body != nil {
		defer response.Body.Close()
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("Error reading individual data: %s\n", err.Error())
		return individualData
	}

	// Parse JSON data from the response body to extract the datapoints we want
	var userData IndividualJSONData
	err = json.Unmarshal(body, &userData)

	if err != nil {
		log.Fatalf("Error parsing individual data: %s\n", err.Error())
		return individualData
	}

	// Calculate the total miles and chains from total distance
	distance := userData.Totals.Distance
	distanceFloat, _ := strconv.ParseFloat(distance, 64)
	miles := int(distanceFloat)
	chains := math.Round((distanceFloat - float64(miles)) * 80)

	individualData.Miles = miles
	individualData.Chains = int(chains)

	return individualData
}

func updateMiles(config Config) {
	// Get the current year
	year := time.Now().Year()

	// Determine which mode is selected
	if config.Mode == "league" {
		// Fetch data
		leagueData := getLeagueData(config)

		// Build the top message

		topMessage := config.Messages.TopLeagueMessage
		topMessage = strings.ReplaceAll(topMessage, "{leagueName}", leagueData.Name)
		topMessage = strings.ReplaceAll(topMessage, "{place}", strconv.Itoa(leagueData.CurrentUser.Position)+getPositionSuffix(leagueData.CurrentUser.Position))
		topMessage = strings.ReplaceAll(topMessage, "{mi}", strconv.Itoa(leagueData.CurrentUser.Miles))
		topMessage = strings.ReplaceAll(topMessage, "{ch}", strconv.Itoa(leagueData.CurrentUser.Chains))
		topMessage = strings.ReplaceAll(topMessage, "{year}", strconv.Itoa(year))

		// Build the bottom message
		bottomMessage := config.Messages.BottomLeagueMessage
		bottomMessage = strings.ReplaceAll(bottomMessage, "{leagueName}", leagueData.Name)
		bottomMessage = strings.ReplaceAll(bottomMessage, "{place}", strconv.Itoa(leagueData.CurrentUser.Position)+getPositionSuffix(leagueData.CurrentUser.Position))
		bottomMessage = strings.ReplaceAll(bottomMessage, "{mi}", strconv.Itoa(leagueData.CurrentUser.Miles))
		bottomMessage = strings.ReplaceAll(bottomMessage, "{ch}", strconv.Itoa(leagueData.CurrentUser.Chains))
		bottomMessage = strings.ReplaceAll(bottomMessage, "{year}", strconv.Itoa(year))

		activity := client.Activity{
			State:   bottomMessage,
			Details: topMessage,

			LargeImage: config.ImageKeys.Large,
			SmallImage: config.ImageKeys.Small,

			Buttons: []*client.Button{},
		}

		if config.Buttons.UserLink.Visible {
			activity.Buttons = append(activity.Buttons, &client.Button{
				Label: config.Buttons.UserLink.Message,
				Url:   config.RailmilesUrl,
			})
		}

		if config.Buttons.LeagueLink.Visible {
			activity.Buttons = append(activity.Buttons, &client.Button{
				Label: config.Buttons.LeagueLink.Message,
				Url:   ("https://my.railmiles.me/leagues/" + strconv.Itoa(config.League)),
			})
		}

		// Update the Rich Presence
		err := client.SetActivity(activity)

		if err != nil {
			log.Fatalf("Error updating Rich Presence: %s\n", err.Error())
			return
		}
	} else if config.Mode == "solo" {
		// Fetch data
		individualData := getIndividualData(config)

		// Build the top message
		topMessage := config.Messages.TopSoloMessage
		topMessage = strings.ReplaceAll(topMessage, "{year}", strconv.Itoa(year))
		topMessage = strings.ReplaceAll(topMessage, "{mi}", strconv.Itoa(individualData.Miles))
		topMessage = strings.ReplaceAll(topMessage, "{ch}", strconv.Itoa(individualData.Chains))

		// Build the bottom message
		bottomMessage := config.Messages.BottomSoloMessage
		bottomMessage = strings.ReplaceAll(bottomMessage, "{year}", strconv.Itoa(year))
		bottomMessage = strings.ReplaceAll(bottomMessage, "{mi}", strconv.Itoa(individualData.Miles))
		bottomMessage = strings.ReplaceAll(bottomMessage, "{ch}", strconv.Itoa(individualData.Chains))

		activity := client.Activity{
			State:   bottomMessage,
			Details: topMessage,

			LargeImage: config.ImageKeys.Large,
			SmallImage: config.ImageKeys.Small,

			Buttons: []*client.Button{},
		}

		if config.Buttons.UserLink.Visible {
			activity.Buttons = append(activity.Buttons, &client.Button{
				Label: config.Buttons.UserLink.Message,
				Url:   config.RailmilesUrl,
			})
		}

		// Update the Rich Presence
		err := client.SetActivity(activity)

		if err != nil {
			log.Fatalf("Error updating Rich Presence: %s\n", err.Error())
			return
		}
	} else {
		log.Fatalf("Invalid mode: %s\n", config.Mode)
		return
	}
}

func trayReady(config Config) {
	systray.SetTemplateIcon(icon.Data, icon.Data)
	systray.SetTitle("RailMiles Rich Presence")
	systray.SetTooltip("RailMiles Rich Presence")

	forceUpdateBtn := systray.AddMenuItem("Update Miles", "Manually updates your RailMiles")

	btnQuit := systray.AddMenuItem("Close", "Close Rich Presence")
	go func() {
		<-btnQuit.ClickedCh
		fmt.Println("Attempting to close...")
		systray.Quit()
		fmt.Println("Closed successfully")
	}()

	for {
		select {
		case <-forceUpdateBtn.ClickedCh:
			forceUpdateBtn.Disable()
			updateMiles(config)
			time.Sleep(10 * time.Second)
			forceUpdateBtn.Enable()
		}
	}
}

// Entrypoint
func main() {
	// Load the configuration file
	log.Println("Loading configuration file...")
	var config Config

	configFile, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Error reading configuration file: %s\n", err.Error())
		return
	}

	// Unmarshal the configuration file
	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		log.Fatalf("Error parsing configuration file: %s\n", err.Error())
		return
	}

	// Create the Rich Presence client
	log.Println("Creating Rich Presence client...")
	err = client.Login(config.ApplicationId)
	if err != nil {
		log.Fatalf("Error initialising Rich Presence client: %s\n", err.Error())
		return
	}

	// Create system tray icon
	systray.Run(func() { trayReady(config) }, func() {})

	// Update the miles
	updateMiles(config)

	// Start cron loop to update miles
	c := cron.New()
	c.AddFunc("0 15,45 * * * *", func() {
		log.Println("Updating miles...")
		updateMiles(config)
	})

	go c.Start()

	// Wait for signal to exit
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, os.Kill)
	<-sig
	systray.Quit()
}

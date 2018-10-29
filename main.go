package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/marni/goigc"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

//MetaInfo for easy metainfo use
type MetaInfo struct {
	Uptime  string `json:"uptime"`
	Info    string `json:"info"`
	Version string `json:"version"`
}

//Track stores data about the track
type Track struct {
	ID          string
	HDate       time.Time `json:"H_Date"`
	Pilot       string    `json:"pilot"`
	Glider      string    `json:"glider"`
	GliderID    string    `json:"glider_id"`
	TrackLength float64   `json:"track_length"`
	URL         string    `json:"track_src_url"`
	TimeStamp bson.ObjectId
}
//Ticker stores info used for ticker
type Ticker struct {
	TLatest    bson.ObjectId `json:"t_latest"`
	TStart     bson.ObjectId `json:"t_start"`
	TStop      bson.ObjectId `json:"t_stop"`
	Tracks     []string      `json:"tracks"`
	Processing string 	 `json:"processing"`
}

//Webhook stores webhook info
type Webhook struct {
	ID       string
	URL      string `json:"webhookURL"`
	Value    int    `json:"minTriggerValue"`
	TrackAdd int

}

//WebhookMessage stores data for the webhook to send
type WebhookMessage struct {
	TLatest    bson.ObjectId `json:"t_latest"`
	Tracks     []string      `json:"tracks"`
	Processing string`json:"processing"`
}

type WebHookSend struct {
	Message WebhookMessage `json:"text"`
}

var urlAmount int
var webhookAmount int
var clockSaved int
var deletedTracks int
var timeStart time.Time

//The variables here are consts in another file not added to github for safety reasons
var trackDataBase trackDB
//Same goes for this, consts stored in another file for security
var webhookDataBase webhookDB

func init() {
	urlAmount = 0
	webhookAmount = 0
	clockSaved = 0
	deletedTracks = 0
	timeStart = time.Now()
	trackDataBase = trackDB{DBURL, DBName, DBCollection}
	webhookDataBase = webhookDB{WebhookURL, WebhookName, WebhookCollection}
}

//If bad link is provided, error message will be shown
func error404(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

//Bad Request error message function as it is used many times
func error400(w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
}

//Errorcheck to check if there was an error with string
func errorCheck(val string) (string, error) {
	return val, nil
}

//Calculate the time passed since server for x seconds
func calcDuration() string {

	now := time.Now()
	now.Format(time.RFC3339)
	timeStart.Format(time.RFC3339)

	return now.Sub(timeStart).String()

}

func calcDistance(track igc.Track) float64 {
	trackdistance := 0.0

	for i := 0; i < len(track.Points)-1; i++ {
		trackdistance += track.Points[i].Distance(track.Points[i+1])
	}
	return trackdistance
}

//shows the meta information about the API
func getMetaInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	parts := strings.Split(r.URL.Path, "/")

	if len(parts) != 4 {
		error404(w, r)
		return
	}

	upTime := strings.Split(calcDuration(), ".")
	meta := MetaInfo{upTime[0], Info, Version}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		panic(err)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(metaJSON)
}

//registers track and returns array of all trackIDs depending on POST and GET
func manageTrack(w http.ResponseWriter, r *http.Request) {

	//Register a new track
	if r.Method == "POST" {
		w.Header().Set("Content-Type", "application/json")
		if r.Body == nil {
			error400(w)
			return
		}

		var trackURL string

		err := json.NewDecoder(r.Body).Decode(&trackURL)

		if err != nil {
			error400(w)
			return
		}

		track, err := igc.ParseLocation(trackURL)
		if err != nil {
			error400(w)
			return
		}
		urlAmount++
		newID := "igc" + strconv.Itoa(urlAmount)

		ID, err := errorCheck(newID)
		if err != nil {
			error400(w)
			return
		}

		newTrack := Track{
			ID,
			track.Date,
			track.Pilot,
			track.GliderType,
			track.GliderID,
			calcDistance(track),
			trackURL,
		bson.NewObjectIdWithTime(time.Now())}


		trackDataBase.Add(newTrack)

		addJSON, err := json.Marshal(newTrack.ID)
		if err != nil {
			error400(w)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(addJSON)

		sendWebhook(w)

	} else if r.Method == "GET" { //returns all the tracks ids
		w.Header().Set("Content-Type", "application/json")

		total := trackDataBase.Count()

		Response := []string{}

		for i := deletedTracks + 1; i <= total; i++ {
			tempTrack, ok := trackDataBase.Get("igc" + strconv.Itoa(i))
			if !ok {
				error400(w)
				return
			}
			Response = append(Response, tempTrack.ID)
		}

		IDJSON, err := json.Marshal(Response)
		if err != nil {
			error400(w)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(IDJSON)

	} else {
		error400(w)
	}
}

//Gives all the information about a certain track by given ID
func getTrackByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	parts := strings.Split(r.URL.Path, "/")
	track := parts[len(parts)-1]
	if track != "" {

		tempTrack, ok := trackDataBase.Get(track)

		if !ok {
			error400(w)
			return
		}

		trackJSON, err := json.Marshal(tempTrack)

		if err != nil {
			error400(w)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(trackJSON)

	} else {
		error404(w, r)
	}

}

//Given id and field, returns the field of given id
func getTrackField(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")

	id := parts[len(parts)-2]
	field := parts[len(parts)-1]

	if id != "" && field != "" {

		tempTrack, ok := trackDataBase.Get(id)

		if !ok {
			error400(w)
		}

		switch field {
		case "pilot":
			Response := tempTrack.Pilot

			response, err := errorCheck(Response)
			if err != nil {
				error400(w)
				return
			}
			fmt.Fprint(w, response)
		case "glider":
			Response := tempTrack.Glider

			response, err := errorCheck(Response)
			if err != nil {
				error400(w)
				return
			}
			fmt.Fprint(w, response)
		case "glider_id":
			Response := tempTrack.GliderID
			response, err := errorCheck(Response)
			if err != nil {
				error400(w)
				return
			}
			fmt.Fprint(w, response)
		case "track_length":
			fmt.Fprint(w, tempTrack.TrackLength)
		case "H_date":
			Response := tempTrack.HDate.String()

			response, err := errorCheck(Response)
			if err != nil {
				error400(w)
				return
			}
			fmt.Fprint(w, response)
		case "track_src_url":
			Response := tempTrack.URL

			response, err := errorCheck(Response)
			if err != nil {
				error400(w)
				return
			}
			fmt.Fprint(w, response)
		}
	} else {
		error404(w, r)
	}
}

func redirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/paragliding/api/", http.StatusSeeOther)
}

func tickerLast(w http.ResponseWriter, r *http.Request) {
	if urlAmount == 0 {
		error400(w)
		return
	}
	tempTimeStamp, ok := trackDataBase.Get("igc" + strconv.Itoa(urlAmount))
	if !ok {
		error400(w)
		return
	}
	fmt.Fprint(w, tempTimeStamp.TimeStamp)
}

func ticker(w http.ResponseWriter, r *http.Request) {
	processStart := time.Now().UnixNano() / int64(time.Millisecond)
	arrayCap := 5

	var startTime bson.ObjectId
	var stopTime bson.ObjectId

	tracks := []string{}

	for i := deletedTracks + 1; i <= arrayCap + deletedTracks; i++ {
		tempTimeStamp, ok := trackDataBase.Get("igc" + strconv.Itoa(i))
		if !ok {
			break
		}
		tracks = append(tracks, tempTimeStamp.ID)

		if i == 1 {
			startTime = tempTimeStamp.TimeStamp
		}
		//Defined here many times in case there are less than 5 tracks
		stopTime = tempTimeStamp.TimeStamp
	}

	trackTotal := trackDataBase.Count()
	lastTimeStamp, ok := trackDataBase.Get("igc" + strconv.Itoa(trackTotal))
	if !ok {
		error400(w)
		return
	}

	latest := lastTimeStamp.TimeStamp

	process := (time.Now().UnixNano() / int64(time.Millisecond)) - processStart
	processString := strconv.FormatInt(process, 10)

	response := Ticker{latest, startTime, stopTime, tracks, processString}

	tickerJSON, err := json.Marshal(response)
	if err != nil {
		error400(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(tickerJSON)
}

func tickerTimeStamp(w http.ResponseWriter, r *http.Request) {
	processStart := time.Now().UnixNano() / int64(time.Millisecond)
	arrayCap := 5

	parts := strings.Split(r.URL.Path, "/")
	stamp := bson.ObjectIdHex(parts[len(parts)-1])

	var startTime bson.ObjectId
	var stopTime bson.ObjectId

	low := 1

	for {
		tempTimeStamp, ok := trackDataBase.Get("igc" + strconv.Itoa(low))
		if !ok {
			error400(w)
			return
		}

		if stamp > tempTimeStamp.TimeStamp {
			break
		}
		low++
	}

	tracks := []string{}

	for i := low; i <= arrayCap+low; i++ {
		tempTimeStamp, ok := trackDataBase.Get("igc" + strconv.Itoa(i))
		if !ok {
			break
		}
		tracks = append(tracks, tempTimeStamp.ID)

		if i == 1 {
			startTime = tempTimeStamp.TimeStamp
		}
		//Defined here many times in case there are less than 5 tracks
		stopTime = tempTimeStamp.TimeStamp
	}

	trackTotal := trackDataBase.Count()
	lastTimeStamp, ok := trackDataBase.Get("igc" + strconv.Itoa(trackTotal))
	if !ok {
		error400(w)
		return
	}

	latest := lastTimeStamp.TimeStamp

	process := (time.Now().UnixNano() / int64(time.Millisecond)) - processStart
	processString := strconv.FormatInt(process, 10)

	response := Ticker{latest, startTime, stopTime, tracks, processString}

	tickerJSON, err := json.Marshal(response)
	if err != nil {
		error400(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(tickerJSON)
}

func newWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		error400(w)
	}

	var newWebhook Webhook

	err := json.NewDecoder(r.Body).Decode(&newWebhook)
	if err != nil {
		error400(w)
		return
	}

	if newWebhook.Value == 0 {
		newWebhook.Value = 1
	}

	webhookAmount++
	newWebhook.ID = strconv.Itoa(webhookAmount)
	newWebhook.TrackAdd = urlAmount

	webhookDataBase.Add(newWebhook)

	fmt.Fprintf(w, newWebhook.ID)
}

func sendWebhook(w http.ResponseWriter) {
	processStart := time.Now().UnixNano() / int64(time.Millisecond)

	hooks := webhookDataBase.Count()
	if hooks == 0 {
		return
	}

	for i := deletedTracks + 1; i <= hooks + deletedTracks; i++ {
		tempWH, ok := webhookDataBase.Get(strconv.Itoa(i))
		if !ok {
			error400(w)
			return
		}
		if tempWH.TrackAdd + tempWH.Value == urlAmount + deletedTracks {
			tempTimeStamp, ok := trackDataBase.Get("igc" + strconv.Itoa(urlAmount))
			if !ok {
				error400(w)
				return
			}
			tracks := []string{}
			for i := tempWH.TrackAdd + 1; i <= urlAmount + deletedTracks; i++ {
				track, ok := trackDataBase.Get("igc" + strconv.Itoa(i))
				if !ok {
					error400(w)
					return
				}
				tracks = append(tracks, track.ID)
			}

			process := (time.Now().UnixNano() / int64(time.Millisecond)) - processStart
			processString := strconv.FormatInt(process, 10)

			messageBody := WebhookMessage{tempTimeStamp.TimeStamp, tracks, processString}

			message := WebHookSend{messageBody}

			messageJSON, err := json.Marshal(message)

			if err != nil {
				error400(w)
				return
			}

			tempWH.TrackAdd = urlAmount + deletedTracks
			ok = webhookDataBase.Delete("igc" + strconv.Itoa(i))
			if !ok {
				error400(w)
				return
			}

			webhookDataBase.Add(tempWH)

			resp, err := http.Post(tempWH.URL, "application/json", bytes.NewBuffer(messageJSON))

			var result map[string]interface{}

			json.NewDecoder(resp.Body).Decode(&result)

			log.Println(result)
			log.Println(result["data"])
		}
	}
}

func manageWebhook(w http.ResponseWriter, r *http.Request) {

	parts := strings.Split(r.URL.Path, "/")
	ID := parts[len(parts)-1]

	if r.Method == "GET" {
		tempWH, ok := webhookDataBase.Get(ID)
		if !ok {
			error400(w)
			return
		}

		resp, err := json.Marshal(tempWH)
		if err != nil {
			error400(w)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(resp)
	} else if r.Method == "DELETE" {
		tempWH, ok := webhookDataBase.Get(ID)
		if !ok {
			error400(w)
			return
		}

		ok = webhookDataBase.Delete(ID)
		if !ok {
			error400(w)
			return
		}

		resp, err := json.Marshal(tempWH)
		if err != nil {
			error400(w)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(resp)
	}
}

func clockTrigger() {
	if clockSaved < urlAmount + deletedTracks {
		processStart := time.Now().UnixNano() / int64(time.Millisecond)

		 tracks := []string{}
		 var tLate bson.ObjectId

		for i := clockSaved + 1; i <= urlAmount + deletedTracks; i++ {
			tempTrack, ok := trackDataBase.Get("igc" + strconv.Itoa(i))
			if !ok {
				panic(ok)
			}
			tracks = append(tracks, tempTrack.ID)

			if i == urlAmount + deletedTracks {
				tLate = tempTrack.TimeStamp
			}
		}

		process := (time.Now().UnixNano() / int64(time.Millisecond)) - processStart
		processString := strconv.FormatInt(process, 10)
		messageBody := WebhookMessage{tLate,tracks,processString}

		message := WebHookSend{messageBody}

		messageJSON, err := json.Marshal(message)
		if err != nil {
			panic(err)
		}

		//SlackURL is hidden in another file not posted to github to ensure no spam to it
		resp, err := http.Post(SlackURL, "application/json", bytes.NewBuffer(messageJSON))

		var result map[string]interface{}

		json.NewDecoder(resp.Body).Decode(&result)

		log.Println(result)
		log.Println(result["data"])
	}
}

func adminGet(w http.ResponseWriter, r *http.Request) {
	count := trackDataBase.Count()
	fmt.Fprint(w, count)
}

func adminDelete(w http.ResponseWriter, r *http.Request) {
	count, ok := trackDataBase.Delete()
	if !ok {
		error400(w)
		return
	}
	fmt.Fprint(w, count)
}

//Runs the application
func main() {
	trackDataBase.Init()
	webhookDataBase.Init()
	router := mux.NewRouter()

	//Clock causes the program to shut down
	//clock := time.NewTimer(10 * time.Minute)
	//<-clock.C
	//clockTrigger()

	router.HandleFunc("/paragliding/api/track/", manageTrack)
	router.HandleFunc("/paragliding/api/", getMetaInfo)
	router.HandleFunc("/paragliding/api/track/{[0-9A-Za-z]+}", getTrackByID)
	router.HandleFunc("/paragliding/api/track/{[0-9]+}/{[A-Za-z]+}", getTrackField)
	router.HandleFunc("/paragliding/", redirect)
	router.HandleFunc("/paragliding/api/ticker/latest", tickerLast)
	router.HandleFunc("/paragliding/api/ticker/", ticker)
	router.HandleFunc("/paragliding/api/ticker/{[0-9A-Za-z]}", tickerTimeStamp)
	router.HandleFunc("/paragliding/api/webhook/new_track/", newWebhook)
	router.HandleFunc("/paragliding/api/webhook/new_track/{[0-9A-Za-z]}", manageWebhook)
	router.HandleFunc("/UnexpectedURL/admin/api/tracks_count", adminGet)
	router.HandleFunc("/UnexpectedURL/admin/api/tracks", adminDelete)
	router.HandleFunc("/", error404)
	http.ListenAndServe(":"+os.Getenv("PORT"), router)
}

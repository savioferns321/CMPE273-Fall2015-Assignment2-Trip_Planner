package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/drone/routes"
)

type locationStruct struct {
	Address    string `json:"address"`
	City       string `json:"city"`
	Coordinate struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"coordinate"`
	ID    bson.ObjectId `json:"id" bson:"_id,omitempty"`
	Name  string        `json:"name"`
	State string        `json:"state"`
	Zip   string        `json:"zip"`
}

type GoogleLocationStruct struct {
	Results []struct {
		AddressComponents []struct {
			LongName  string   `json:"long_name"`
			ShortName string   `json:"short_name"`
			Types     []string `json:"types"`
		} `json:"address_components"`
		FormattedAddress string `json:"formatted_address"`
		Geometry         struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
			LocationType string `json:"location_type"`
			Viewport     struct {
				Northeast struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"northeast"`
				Southwest struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"southwest"`
			} `json:"viewport"`
		} `json:"geometry"`
		PartialMatch bool     `json:"partial_match"`
		PlaceID      string   `json:"place_id"`
		Types        []string `json:"types"`
	} `json:"results"`
	Status string `json:"status"`
}

func addLocation(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var t locationStruct
	err := decoder.Decode(&t)
	if err != nil {
		panic("Some error in decoding the JSON")
	}
	//TODO Hit Google's API and retrieve the co-ordinates, then save this struct to a MongoDB instance and retrieve the auto-generated ID.
	requestUrl := strings.Join([]string{t.Address, t.City, t.State, t.Zip}, "+")
	requestUrl = strings.Replace(requestUrl, " ", "%20", -1)

	var buffer bytes.Buffer
	buffer.WriteString("http://maps.google.com/maps/api/geocode/json?address=")
	buffer.WriteString(requestUrl)
	buffer.WriteString("&sensor=false")

	url := buffer.String()
	fmt.Println("Url is ", url)

	res, err := http.Get(url)
	if err != nil {
		//w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{    "error": "Unable to parse data from Google. Error at res, err := http.Get(url) -- line 75"}`))
		panic(err.Error())
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		//w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{    "error": "Unable to parse data from Google. body, err := ioutil.ReadAll(res.Body) -- line 84"}`))
		panic(err.Error())
	}
	var googleLocation GoogleLocationStruct

	err = json.Unmarshal(body, &googleLocation)
	if err != nil {
		//w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{    "error": "Unable to unmarshal Google data. body, 	err = json.Unmarshal(body, &googleLocation) -- line 94"}`))
		panic(err.Error())
	}

	//TODO Prepare a response which will return the ID and the details of the JSON in JSON format.
	t.Coordinate.Lat = googleLocation.Results[0].Geometry.Location.Lat
	t.Coordinate.Lng = googleLocation.Results[0].Geometry.Location.Lng
	t.ID = bson.NewObjectId()
	//TODO Set the ID as the auto generated ID from MongoDB
	maxWait := time.Duration(20 * time.Second)
	session, err := mgo.DialWithTimeout("mongodb://savioferns321:mongodb123@ds041633.mongolab.com:41633/savio_mongo", maxWait)
	if err != nil {
		fmt.Println("Unable to connect to MongoDB")
		panic(err)
	}
	defer session.Close()
	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("savio_mongo").C("addresses")
	err = c.Insert(bson.M{"_id": t.ID, "name": t.Name, "address": t.Address, "city": t.City, "state": t.State, "zip": t.Zip, "coordinate": bson.M{"lat": t.Coordinate.Lat, "lng": t.Coordinate.Lng}})
	if err != nil {
		fmt.Println("Error at line 124 ---- c := session.DB(tripplannerdb).C(addresses)")
		log.Fatal(err)
	}

	//TODO Store the output JSON into MongoDB with the ID as its key
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	outputJson, err := json.Marshal(t)
	if err != nil {
		//w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{    "error": "Unable to marshal response. body, 	outputJson, err := json.Marshal(t) -- line 110"}`))
		panic(err.Error())
	}
	w.Write(outputJson)
}

func findLocation(w http.ResponseWriter, r *http.Request) {

	locationId := r.URL.Query().Get(":locationId")
	fmt.Println("Location ID is : ", locationId)
	maxWait := time.Duration(20 * time.Second)
	session, err := mgo.DialWithTimeout("mongodb://savioferns321:mongodb123@ds041633.mongolab.com:41633/savio_mongo", maxWait)
	if err != nil {
		fmt.Println("Unable to connect to MongoDB")
		panic(err)
	}
	defer session.Close()
	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	var result locationStruct

	c := session.DB("savio_mongo").C("addresses")
	err = c.Find(bson.M{"_id": bson.ObjectIdHex(locationId)}).One(&result)
	if err != nil {
		fmt.Println("err = c.Find(bson.M{\"id\": bson.M{\"$oid\": t}}).One(&result)")
		log.Fatal(err)
	}

	//Returning the result to user
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	outputJson, err := json.Marshal(result)
	if err != nil {
		//w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{    "error": "Unable to marshal response. body, 	outputJson, err := json.Marshal(t) -- line 110"}`))
		panic(err.Error())
	}
	w.Write(outputJson)

}

func updateLocation(w http.ResponseWriter, r *http.Request) {
	locationId := r.URL.Query().Get(":locationId")
	fmt.Println("Received location ID ", locationId)
	decoder := json.NewDecoder(r.Body)
	var t locationStruct
	err := decoder.Decode(&t)
	if err != nil {
		panic("Some error in decoding the JSON")
	}
	t.ID = bson.ObjectIdHex(locationId)

	//If location values are not null then update the co-ordinates from Google API
	var requestUrl bytes.Buffer
	if len(t.Address) > 0 {
		requestUrl.WriteString(t.Address + "+")
	}
	if len(t.City) > 0 {
		requestUrl.WriteString(t.City + "+")
	}
	if len(t.State) > 0 {
		requestUrl.WriteString(t.State + "+")
	}
	if len(t.Zip) > 0 {
		requestUrl.WriteString(t.Zip)
	}

	if requestUrl.Len() > 0 {
		//The Location needs to be changed via Google API
		requestStr := strings.Replace(requestUrl.String(), " ", "%20", -1)
		var buffer bytes.Buffer
		buffer.WriteString("http://maps.google.com/maps/api/geocode/json?address=")
		buffer.WriteString(requestStr)
		buffer.WriteString("&sensor=false")
		url := buffer.String()
		fmt.Println("Url is ", url)

		res, err := http.Get(url)
		if err != nil {
			//w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{    "error": "Unable to parse data from Google. Error at res, err := http.Get(url) -- line 75"}`))
			panic(err.Error())
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			//w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{    "error": "Unable to parse data from Google. body, err := ioutil.ReadAll(res.Body) -- line 84"}`))
			panic(err.Error())
		}
		var googleLocation GoogleLocationStruct

		err = json.Unmarshal(body, &googleLocation)
		if err != nil {
			//w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{    "error": "Unable to unmarshal Google data. body, 	err = json.Unmarshal(body, &googleLocation) -- line 94"}`))
			panic(err.Error())
		}

		//TODO Prepare a response which will return the ID and the details of the JSON in JSON format.
		t.Coordinate.Lat = googleLocation.Results[0].Geometry.Location.Lat
		t.Coordinate.Lng = googleLocation.Results[0].Geometry.Location.Lng
	}

	//Perform the update
	maxWait := time.Duration(20 * time.Second)
	session, err := mgo.DialWithTimeout("mongodb://savioferns321:mongodb123@ds041633.mongolab.com:41633/savio_mongo", maxWait)
	if err != nil {
		fmt.Println("Unable to connect to MongoDB")
		panic(err)
	}
	defer session.Close()
	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("savio_mongo").C("addresses")
	err = c.Update(bson.M{"_id": bson.ObjectIdHex(locationId)}, t)
	if err != nil {
		fmt.Println("	Line 248 : err = c.Update(bson.M{id: bson.M{$oid: locationId}}, t)")
		log.Fatal(err)
	}

	//Prepare and write the response
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	outputJson, err := json.Marshal(t)
	if err != nil {
		//w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{    "error": "Unable to marshal response. body, 	outputJson, err := json.Marshal(t) -- line 110"}`))
		panic(err.Error())
	}
	w.Write(outputJson)
	fmt.Println("Update done successfully!")

}

func deleteLocation(w http.ResponseWriter, r *http.Request) {

	locationId := r.URL.Query().Get(":locationId")
	fmt.Println("Location ID is : ", locationId)
	maxWait := time.Duration(20 * time.Second)
	session, err := mgo.DialWithTimeout("mongodb://savioferns321:mongodb123@ds041633.mongolab.com:41633/savio_mongo", maxWait)
	if err != nil {
		fmt.Println("Unable to connect to MongoDB")
		panic(err)
	}
	defer session.Close()
	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("savio_mongo").C("addresses")
	err = c.Remove(bson.M{"_id": bson.ObjectIdHex(locationId)})
	if err != nil {
		fmt.Println("err = c.Find(bson.M{\"id\": bson.M{\"$oid\": t}}).One(&result)")
		log.Fatal(err)
	}

	//Returning the result to user
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		//w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{    "error": "Unable to marshal response. body, 	outputJson, err := json.Marshal(t) -- line 110"}`))
		panic(err.Error())
	}
	w.Write([]byte(`{"result": "Delete operation done successfully."}`))

}

func main() {
	mux := routes.New()
	mux.Post("/locations/", addLocation)
	mux.Get("/locations/:locationId", findLocation)
	mux.Put("/locations/:locationId", updateLocation)
	mux.Del("/locations/:locationId", deleteLocation)
	http.Handle("/", mux)
	http.ListenAndServe(":8088", nil)
}

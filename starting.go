package main

import (
	"fmt"
	"log"
	"net/http"
	"io/ioutil"
	"os"
	"encoding/json"
	"sort"
	"github.com/gorilla/mux"
)

//global variable to be accessed from different handlers/functions
var data BaseData

//Error struct to send a errors in json format
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
type Type struct {
	// Name of the type
	Name string `json:"name"`
	// The effective types, damage multiplize 2x
	EffectiveAgainst []string `json:"effectiveAgainst"`
	// The weak types that against, damage multiplize 0.5x
	WeakAgainst []string `json:"weakAgainst"`
}

type Pokemon struct {
	Number         string   `json:"Number"`
	Name           string   `json:"Name"`
	Classification string   `json:"Classification"`
	TypeI          []string `json:"Type I"`
	TypeII         []string `json:"Type II,omitempty"`
	Weaknesses     []string `json:"Weaknesses"`
	FastAttackS    []string `json:"Fast Attack(s)"`
	Weight         string   `json:"Weight"`
	Height         string   `json:"Height"`
	Candy struct {
		Name     string `json:"Name"`
		FamilyID int    `json:"FamilyID"`
	} `json:"Candy"`
	NextEvolutionRequirements struct {
		Amount int    `json:"Amount"`
		Family int    `json:"Family"`
		Name   string `json:"Name"`
	} `json:"Next Evolution Requirements,omitempty"`
	NextEvolutions []struct {
		Number string `json:"Number"`
		Name   string `json:"Name"`
	} `json:"Next evolution(s),omitempty"`
	PreviousEvolutions []struct {
		Number string `json:"Number"`
		Name   string `json:"Name"`
	} `json:"Previous evolution(s),omitempty"`
	SpecialAttacks      []string `json:"Special Attack(s)"`
	BaseAttack          int      `json:"BaseAttack"`
	BaseDefense         int      `json:"BaseDefense"`
	BaseStamina         int      `json:"BaseStamina"`
	CaptureRate         float64  `json:"CaptureRate"`
	FleeRate            float64  `json:"FleeRate"`
	BuddyDistanceNeeded int      `json:"BuddyDistanceNeeded"`
}

// Move is an attack information. The
type Move struct {
	// The ID of the move
	ID int `json:"id"`
	// Name of the attack
	Name string `json:"name"`
	// Type of attack
	Type string `json:"type"`
	// The damage that enemy will take
	Damage int `json:"damage"`
	// Energy requirement of the attack
	Energy int `json:"energy"`
	// Dps is Damage Per Second
	Dps float64 `json:"dps"`
	// The duration
	Duration int `json:"duration"`
}

// BaseData is a struct for reading data.json
type BaseData struct {
	Types    []Type    `json:"types"`
	Pokemons []Pokemon `json:"pokemons"`
	Moves    []Move    `json:"moves"`
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("/list url:", r.URL)
	//structure of response
	var resp = map[string]interface{}{}
	//try to get parameters from url
	givenTypes, ctype := r.URL.Query()["type"]

	if !ctype {
		resp["Error"] = Error{Code: 2, Message: "Type is missing"}
		writeResponse(w, resp)
		return
	} else {
		//check the types of pokemons with given type
		givenType := string(givenTypes[0])
		//slice to store desired Pokemons
		var arr []Pokemon
		for _, pokemon := range data.Pokemons {
			condType2 := true
			condPrev := true
			condNext := true
			//controls to avoid panics
			if len(pokemon.TypeII) == 0 {
				condType2 = false
			}
			if len(pokemon.PreviousEvolutions) == 0 {
				condPrev = false
			}
			if len(pokemon.NextEvolutions) == 0 {
				condNext = false
			}
			//All these controls for finding a pokemons of given type.
			//Task says that I should consider the next and previous evolutions and pokemon's second types to find proper pokemons,
			if pokemon.TypeI[0] == givenType ||
				(condType2 && pokemon.TypeII[0] == givenType) ||
				(condPrev && checkContains(pokemon.PreviousEvolutions, givenType)) ||
				(condNext && checkContains(pokemon.NextEvolutions, givenType)) {
				arr = append(arr, pokemon)
			}
		}
		if len(arr) == 0 {
			resp["Error"] = Error{Code: 5, Message: "Invalid type for Pokemons"}
			writeResponse(w, resp)
			return
		}
		// does request have a valid parameter to sort desired type of Pokemons ?
		// if so, call the right function
		// if not, continue and send reponse without sorting
		sortBy, csort := r.URL.Query()["sortby"]
		if csort {
			sortby := sortBy[0]
			if sortby == "BaseStamina" {
				arr = sortSliceByStamina(arr)
			} else if sortby == "BaseAttack" {
				arr = sortSliceByAttack(arr)
			} else if sortby == "BaseDefence" {
				arr = sortSliceByDefense(arr)
			} else {
				resp["Error"] = Error{Code: 3, Message: "Invalid parameter to sort Pokemons"}
				writeResponse(w, resp)
				return
			}
		}
		resp["Response"] = arr
		writeResponse(w, resp)
	}

}

func getHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("/get/{name} url:", r.URL)
	//structure of response
	var resp = map[string]interface{}{}

	params := mux.Vars(r)
	var found = false
	//Search a name for Pokemon, Type or Move
	for _, item := range data.Pokemons {
		if item.Name == params["name"] {
			found = true
			resp["Pokemon"] = item
			writeResponse(w, resp)
			break
		}
	}
	for _, item := range data.Types {
		if item.Name == params["name"] {
			found = true
			resp["Type"] = item
			writeResponse(w, resp)
			break
		}
	}
	for _, item := range data.Moves {
		if item.Name == params["name"] {
			found = true
			resp["Move"] = item
			writeResponse(w, resp)
			break
		}
	}
	//did not find any relevant data, let's send an error
	if !found {
		resp["Error"] = Error{Code: 1, Message: "Invalid request for Pokemon, Type or Move"}
		writeResponse(w, resp)
	}

}

func otherwise(w http.ResponseWriter, r *http.Request) {
	//structure of response
	var resp = map[string]interface{}{}
	resp["Error"] = Error{Code: 0, Message: "You do not know what you are looking for :)"}
	writeResponse(w, resp)

}
func typesHandler(w http.ResponseWriter, r *http.Request) {
	var resp = map[string]interface{}{}
	var slc []Type
	for _, types := range data.Types {
		slc = append(slc, types)
	}
	resp["Response"] = slc
	writeResponse(w, resp)
}

func main() {
	// initialize global variable by json file
	initJson()

	//used a mux to route requests
	router := mux.NewRouter()
	router.HandleFunc("/list", listHandler).Methods("GET")
	router.HandleFunc("/get/{name}", getHandler).Methods("GET")
	router.HandleFunc("/list/types", typesHandler).Methods("GET")
	router.HandleFunc("/{name}", getHandler).Methods("GET")
	router.HandleFunc("/", otherwise).Methods("GET")
	log.Println("starting server on :8080")
	http.ListenAndServe(":8080", router)
}

//Reading a JSON file and storing in a global variable
func initJson() {
	file, err := ioutil.ReadFile("./data.json")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	json.Unmarshal(file, &data)
}

//A function to send a response to the client in json format
//response is a json format to see details better on Firefox or Chrome with JSON Formatter (https://github.com/callumlocke/json-formatter)
func writeResponse(w http.ResponseWriter, response map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	var resp []byte
	resp, _ = json.Marshal(response)
	w.Write(resp)
}

//Sorts given array by BaseStamina
func sortSliceByStamina(ss []Pokemon) []Pokemon {
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].BaseStamina > ss[j].BaseStamina
	})
	return ss
}

//Sorts given array by BaseAttack
func sortSliceByAttack(ss []Pokemon) []Pokemon {
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].BaseAttack > ss[j].BaseAttack
	})
	return ss
}

//Sorts given array by BaseDefense
func sortSliceByDefense(ss []Pokemon) []Pokemon {
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].BaseDefense > ss[j].BaseDefense
	})
	return ss
}

//check whether a struct has a given type or not
func checkContains(data []struct {
	Number string `json:"Number"`
	Name   string `json:"Name"`
}, givenType string) bool {
	for _, item := range data {
		if checkType(item.Name, givenType) {
			return true
			break
		}
	}
	return false
}

//helper func to checkContains to determine next or previous evolutions has the given type
func checkType(pokemonName string, givenType string) bool {
	retValue := false
	condType2 := true
	for _, item := range data.Pokemons {
		if len(item.TypeII) == 0 {
			condType2 = false
		}
		if item.Name == pokemonName {
			retValue = item.TypeI[0] == givenType || (condType2 && item.TypeII[0] == givenType)
		}
	}
	return retValue
}

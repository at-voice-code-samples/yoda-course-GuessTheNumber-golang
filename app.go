package main

import (
	"os"
	"fmt"
	"net/http"
	"math/rand"
	"strconv"
	"sync"
)

type sessionState struct {
	randomNumber, tries int
}

var sessionDb = make(map[string]sessionState)
var sessionDbMutex = &sync.Mutex{}

func main () {
	// Check command-line arguments
	if len(os.Args) != 3 {
		fmt.Println("Usage: ./app <callback-url> <port>")
		os.Exit(0)
	}
	
	http.HandleFunc("/digits", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		sessionId := r.Form.Get("sessionId")
		g,_ := strconv.ParseInt(r.Form.Get("dtmfDigits"), 10, 64)
		guess := int(g)
		sessionDbMutex.Lock()
		session,_ := sessionDb[sessionId]
		sessionDbMutex.Unlock()
		
		if guess > 20 {
			fmt.Fprintf(w, `<Response>
					   <GetDigits timeout='30' finishOnKey='#' callbackUrl='%s'>
					     <Say>%d is above 20. Please pick a number between 0 and 20 followed by hash</Say>
					   </GetDigits>
					   <Say>Sorry, we did not get your input. Goodbye.</Say>
					 </Response>`, os.Args[1]+"/digits", guess)
			return
		}
		
		if session.randomNumber == guess {
			fmt.Fprintf(w, `<Response>
					    <Say>Congratulations, you got it right.</Say>
					</Response>`)
			sessionDbMutex.Lock()
			delete(sessionDb, sessionId)
			sessionDbMutex.Unlock()
		}else{
			if session.tries < 4 {
				var state string
				if session.randomNumber < guess {
					state = "lower"
				}else{
					state = "higher"
				}
				chances := 4 - session.tries
				chancesPlurality := "chances"
				if chances == 1 {
					chancesPlurality = "chance"
				}
				fmt.Fprintf(w, `<Response>
						  <GetDigits timeout='30' finishOnKey='#' callbackUrl='%s'>
						    <Say>The number I am thinking about is %s. You have %d more %s. Guess again.</Say>
						  </GetDigits>
						  <Say>Sorry, we did not get your input. Goodbye.</Say>
						</Response>`,
					os.Args[1]+"/digits", state, chances, chancesPlurality)
				session.tries += 1
				sessionDbMutex.Lock()
				sessionDb[sessionId] = session
				sessionDbMutex.Unlock()
			}else{
				fmt.Fprintf(w, `<Response>
						  <Say>Sorry, you have exhausted your guesses. You lose.</Say>
						</Response>`)
				sessionDbMutex.Lock()
				delete(sessionDb, sessionId)
				sessionDbMutex.Unlock()
			}
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		sessionId := r.Form.Get("sessionId")
		// Check whether the session exists
		sessionDbMutex.Lock()
		if _,exists := sessionDb[sessionId]; exists {
			// Check whether the call is active, clean up if not
			if r.Form.Get("isActive") == "0" {
				delete(sessionDb, sessionId)
			}
		}else{
			// Create a new session and start the game
			newSession := sessionState{randomNumber: rand.Intn(20), tries: 0}
			sessionDb[sessionId] = newSession
			fmt.Fprintf(w, `<Response>
					  <GetDigits timeout='30' finishOnKey='#' callbackUrl='%s'>
					    <Say>Hello there, I am thinking of a number between zero and twenty. Can you guess it within five tries? Enter your guess followed by the hash sign.</Say>
					  </GetDigits>
					  <Say>Sorry, we did not get your input. Goodbye.</Say>
					</Response>`, os.Args[1]+"/digits")
		}
		sessionDbMutex.Unlock()
	})
	http.ListenAndServe(":"+os.Args[2], nil)
}

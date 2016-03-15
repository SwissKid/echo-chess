package main

import (
	"encoding/json"
	"fmt"
	"github.com/malbrecht/chess"
	"github.com/malbrecht/chess/engine/uci"
	"io/ioutil"
	"log"
	"net/http"
	"time"
	"strings"
	"bytes"
	"strconv"
)

var e *uci.Engine
var Accounts map[string]Account

//Program Specific
type Account struct {
	Amazon string //Client ID from amazon
	Board  Game
	Previous string
}

type Game struct {
	Fen        string
	Difficulty string
	LastMove   chess.Move
}

//Request Fields
type RequestMaster struct {
	Version string  `json:"version"`
	Session Session `json:"session"`
	Request Request `json:"request"`
}
type Session struct {
	New         bool              `json:"new"`
	SessionId   string            `json:"sessionId"`
	Attributes  map[string]string `json:"attributes"`
	Application Application       `json:"application"`
	User        User              `json:"user"`
}

type Application struct {
	ApplicationId string
}
type User struct {
	UserId      string
	AccessToken string
}
type Request struct { //I have to combine all the different requests into this. ugh.
	Type      string `json:"type"` //LaunchRequest,IntentRequest,SessionEndedRequest
	Timestamp string `json:"date"`
	RequestId string `json:"requestId"`
	Intent    Intent `json:"intent"`
	Reason    string `json:"reason"`
}
type Intent struct {
	Name  string          `json:"name"`
	Slots map[string]Slot `json:"slots"`
}
type Slot struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

//Response Section
type ResponseMaster struct {
	Version           string                 `json:"version"`
	SessionAttributes map[string]interface{} `json:"sessionAttributes,omitempty"`
	Response          Response               `json:"response"`
}

type Response struct {
	OutputSpeech     *OutputSpeech `json:"outputSpeech,omitempty"`
	Card             *Card         `json:"card,omitempty"`
	Reprompt         *Reprompt     `json:"reprompt,omitempty"`
	ShouldEndSession bool          `json:"shouldEndSession"`
}
type OutputSpeech struct {
	Type string `json:"type,omitempty"` //must omit empty so it doesn't print the whole object
	Text string `json:"text,omitempty"` //must omit empty so it doesn't print the whole object
}
type Card struct {
	Type    string `json:"type,omitempty"` //must omit empty so it doesn't print the whole object
	Title   string `json:"title,omitempty"`
	Content string `json:"content,omitempty"`
}
type Reprompt struct {
	OutputSpeech OutputSpeech `json:"outputSpeech,omitempty"`
}

func moveInList(a chess.Move, list []chess.Move) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
func newgame(userid string, difficulty string){
	var g Game
	board, _ := chess.ParseFen("")
	a :=Accounts[userid]
	g.Fen = board.Fen()
	g.Difficulty = difficulty
	a.Board = g
	Accounts[userid]=a
}
func foo(w http.ResponseWriter, r *http.Request) {
	var l RequestMaster
	var j ResponseMaster
	//var k OutputSpeech
	body, _ := ioutil.ReadAll(r.Body)
	fmt.Println(string(body[:]))
	json.Unmarshal(body, &l)
	// For all responses
	j.Version = "1.0"
	j.Response.ShouldEndSession = true
	switch l.Request.Type{
	    case "LaunchRequest": //"Run Chess"
		j = startSession(l)
	    case "IntentRequest":
		j = continueSession(l)
	}
	b, _ := json.Marshal(j)
	fmt.Println(string(b[:]))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("charset", "UTF-8")
	w.Write(b)
}
func continueSession(l RequestMaster)(j ResponseMaster){
	var k OutputSpeech
	k.Type="PlainText"
	userid := l.Session.User.UserId
	a := Accounts[userid]
	j.Response.ShouldEndSession = false
	switch l.Request.Intent.Name{
	case "Yes":
	    if Accounts[userid].Previous=="NewGame"{
		k.Text = "Please pick a difficulty level between 0 and 20."
		a.Previous="Difficulty"
		Accounts[userid] = a
	    } else if Accounts[userid].Previous=="ContinueGame"{
		move := a.Board.LastMove
		from := move.From.String()
		to := move.To.String()
		k.Text = "Please state your next move. The last move was from " + from + " to " + to + "."
		m := createCard(a.Board)
		j.Response.Card = &m
	    }
	case "NewGame":
		k.Text = "Please pick a difficulty level between 0 and 20."
		a.Previous="Difficulty"
		Accounts[userid] = a
	case "No":
	    a.Previous="Goodbye"
	    k.Text = "Goodbye"
	    m := createCard(a.Board)
	    j.Response.Card = &m
	    j.Response.ShouldEndSession = true
	case "Difficulty":
	    diffValue := l.Request.Intent.Slots["Diff"].Value
	    newgame(userid, diffValue)
	    a.Previous="CreatedGame"
	    k.Text="Created a new game with difficulty " + diffValue + ". Your move!"
	case "Move":
	    k.Text = chessMove(userid, l.Request.Intent.Slots["LocOne"].Value, l.Request.Intent.Slots["LocTwo"].Value)
	}
	j.Response.OutputSpeech = &k
	return j
}
func createCard(g Game)(c Card){
    var buf bytes.Buffer
    move := g.LastMove
    from := move.From.String()
    to := move.To.String()

    c.Title = "State of board after " + from + " to " + to + "."
    c.Type = "Simple"
    board, err := chess.ParseFen(g.Fen)
    if err != nil {
	fmt.Println(err)
    }
    buf.WriteString("1")
    for i, v := range board.Piece{
	if i%8 == 0 && i!=0{
	    num := i/8
	    buf.WriteString( strconv.Itoa(num) + "\n" + strconv.Itoa(num+1))
	}
	if chess.Figurines[v] == '.'{
	    buf.WriteRune(0x2610)
    } else {
	buf.WriteRune(chess.Figurines[v])
    }

    }
    buf.WriteString("8")
    c.Content = buf.String()
    return c
}



func startSession(l RequestMaster)(j ResponseMaster){
	var k OutputSpeech
	k.Type="PlainText"
	j.Response.ShouldEndSession = false
	userid := l.Session.User.UserId
	a := Accounts[userid]
	fmt.Println(Accounts)
	if val, ok := Accounts[userid]; !ok {
	    k.Text = "Hi! You're new here! Want to start a new game?"
	    a.Previous = "NewGame"
	} else {
	    if val.Board.Fen == ""{
		k.Text = "Welcome back! I don't have a stored game for you, would you like to start a new one?"
		a.Previous = "NewGame"
	    } else {
		k.Text = "Welcome Back! Want to continue your game?"
		a.Previous = "ContinueGame"
	    }
	}
	Accounts[userid] = a
	j.Response.OutputSpeech = &k
	return j

}


func chessMove(userid string, pos1 string, pos2 string)(k string){
    var moveS string
    moveS = strings.ToLower(pos1 + pos2)
    moveS = strings.Replace(moveS, " ", "", -1)
    k = "You moved " + moveS + "."
    var a Account
    a = Accounts[userid]
    board, err := chess.ParseFen(a.Board.Fen)
    if err != nil{
	fmt.Println(err)
	return k
    }
    fmt.Println("Their board is at " + board.Fen())
    move, err := board.ParseMove(moveS)
    if err != nil{
	fmt.Println(err)
	k = moveS + " is an Illegal move"
	return k
    }
    board = board.MakeMove(move)
    a.Board.Fen = board.Fen()
    a.Board.LastMove = move
    Accounts[userid] = a
    fmt.Println("Their board is now at " + board.Fen())
    move = calcMove(board, a.Board.Difficulty)
    board = board.MakeMove(move)
    a.Board.Fen = board.Fen()
    a.Board.LastMove = move
    Accounts[userid] = a
    from := move.From.String()
    to := move.To.String()
    fmt.Println("Their board is now at " + board.Fen())
    k = "You moved " + moveS + "."  + " I moved from " + from + " to " + to + "."
    return k
}

func main() {
	Accounts = make(map[string]Account)
	http.HandleFunc("/", foo)
	http.ListenAndServe(":9004", nil)
}

func testchess() {
	var move chess.Move
	var check bool
	var mate bool
	var err error
	board, _ := chess.ParseFen("")
	fmt.Println(board.Fen())
	for mate != true {
		move = calcMove(board, "5")
		move, err = board.ParseMove(move.San(board))
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("%+v \n", move)
		board = board.MakeMove(move)

		fmt.Println(board.Fen())
		check, mate = board.IsCheckOrMate()
		if check {
			fmt.Println("CHECK")
		}
	}
}

func calcMove(board *chess.Board, difficulty string) (move chess.Move) {
	var mov *chess.Move
	var logger *log.Logger //= log.New(stdout, "", log.LstdFlags)
	e, err := uci.Run("stockfish", nil, logger)
	if err != nil {
		fmt.Println(err)
	}
	e.Options()["Skill Level"].Set(difficulty)
	e.SetPosition(board)
	//fmt.Println(test.BestMove())
	for info := range e.SearchDepth(5) {
		if info.Err() != nil {
			log.Printf("%s", info.Err())
		}
		if m, ok := info.BestMove(); ok {
			log.Println("bestmove:", move.San(board))
			mov = &m
		} else if pv := info.Pv(); pv != nil {
			//	    log.Println("pv:", pv)
			//	    log.Println("stats:", info.Stats())
		} else {
			//log.Println("stats:", info.Stats())
		}
	}
	for mov.To == mov.From {
	}
	move = *mov
	fmt.Printf("%+v", e.Options())
	time.Sleep(100 * time.Millisecond)
	return move
}

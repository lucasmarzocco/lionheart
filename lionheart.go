package lionheart

import (
	"bufio"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type User struct {
	Name      string
	Phone     string
	Email     string
	Subtraits map[string]*Trait
	Traits    map[string]*Trait
	Data      map[int]Question
}

type Trait struct {
	Name        string
	RawScore    float64
	NormalScore float64
	Min         float64
}

type Question struct {
	Number      int
	Description string
	Key         int
	Trait       string
	Min         float64
}

type Event struct {
	Id   string `json:"event_id"`
	Type string `json:"event_type"`
	Form Form   `json:"form_response"`
}

type Form struct {
	Id         string     `json:"form_id"`
	Token      string     `json:"token"`
	Landed     string     `json:"landed_at"`
	Submitted  string     `json:"submitted_at"`
	Definition Definition `json:"definition"`
	Answers    []Answer   `json:"answers"`
}

type Definition struct {
	Id     string          `json:"id"`
	Title  string          `json:"title"`
	Fields []QuestionField `json:"fields"`
}

type QuestionField struct {
	Id         string      `json:"id"`
	Title      string      `json:"title"`
	Type       string      `json:"type"`
	Ref        string      `json:"ref"`
	Properties interface{} `json:"properties"`
}

type Answer struct {
	Type   string      `json:"type"`
	Text   string      `json:"text"`
	Email  string      `json:"email"`
	Phone  string      `json:"phone_number"`
	Number int         `json:"number"`
	Field  AnswerField `json:"field"`
}

type AnswerField struct {
	Id   string `json:"id"`
	Type string `json:"type"`
	Ref  string `json:"ref"`
}

func (u *User) LoadQuestionsFromFile() {
	data := make(map[int]Question)

	file, err := os.Open(os.Getenv("TESTFILE"))
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()

		question := strings.Split(text, "|")
		number, _ := strconv.Atoi(question[0])
		key, _ := strconv.Atoi(question[2])
		min, _ := strconv.Atoi(question[4])

		q := Question{
			Number:      number,
			Description: question[1],
			Key:         key,
			Trait:       question[3],
			Min:         float64(min),
		}

		data[number] = q
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	u.Data = data
}

func (u *User) ProcessSubtraits(test []byte) {

	var event Event
	subs := make(map[string]*Trait)

	err := json.Unmarshal(test, &event)
	if err != nil {
		return
	}

	u.Name = event.Form.Answers[0].Text
	u.Email = event.Form.Answers[1].Email
	u.Phone = event.Form.Answers[2].Phone

	for i, ele := range event.Form.Answers[3:] {

		entry := u.Data[i+1]

		if entry.Trait == "" {
			continue
		}

		if _, ok := subs[entry.Trait]; ok {
			subs[entry.Trait].RawScore += float64(entry.Key * ele.Number)
		} else {
			subs[entry.Trait] = &Trait{
				entry.Trait,
				float64(entry.Key * ele.Number),
				0,
				entry.Min,
			}
		}
	}

	u.Subtraits = subs
}

func (u *User) NormalizeSubtraits() {

	for _, v := range u.Subtraits {
		s := 6.25 * (v.RawScore - v.Min)
		m := math.RoundToEven(s)
		v.NormalScore = m
	}
}

func (u *User) ProcessTraits() {

	traits := make(map[string]*Trait)

	for k, v := range u.Subtraits {

		letter := string(k[0])

		if _, ok := traits[letter]; ok {
			traits[letter].RawScore += v.RawScore
		} else {
			traits[letter] = &Trait{
				letter,
				v.RawScore,
				0,
				getMin(letter),
			}
		}
	}

	u.Traits = traits
}

func (u *User) NormalizeTraits() {

	for _, v := range u.Traits {
		s := (100 / float64(96)) * (v.RawScore - v.Min)
		m := math.RoundToEven(s)
		v.NormalScore = m
	}
}

func (u *User) WriteUserData(link string) {

	accountSid := os.Getenv("ACCOUNT_SID")
	token := os.Getenv("TOKEN")
	urlStr := "https://api.twilio.com/2010-04-01/Accounts/" + accountSid + "/Messages.json"

	msgData := url.Values{}
	msgData.Set("To", u.Phone)
	msgData.Set("From", os.Getenv("PHONE"))
	msgData.Set("Body", "Hello! Your Lionheart test results can be found at: " + link)
	msgDataReader := *strings.NewReader(msgData.Encode())

	client := &http.Client{}
	req, err := http.NewRequest("POST", urlStr, &msgDataReader)
	if err != nil {
		panic(err)
	}
	req.SetBasicAuth(accountSid, token)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	client.Do(req)
}

func getMin(letter string) float64 {

	switch letter {
	case "A":
		return -66
	case "C":
		return -36
	case "E":
		return 6
	case "N":
		return -66
	case "O":
		return -78
	}

	return 0
}

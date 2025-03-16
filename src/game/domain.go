package game

type Question struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type Player struct {
	Username              string
	DistinguishedName     string // otherwise known as a DN
	Questions             []Question
	Groups                []string
	CurrentQuestionNumber int
}

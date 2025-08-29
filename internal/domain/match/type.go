package match

import "time"

type Card struct {
	ID      string
	Map     string
	Region  string
	Started time.Time
	Team1   []string
	Team2   []string
	Score1  *int
	Score2  *int
}

package aarhusboligventeliste

import "time"

type Appartment struct {
	ID          string
	Ranks       []RankPair
	CurrentRank int
}

type RankPair struct {
	Time time.Time
	Rank int
}

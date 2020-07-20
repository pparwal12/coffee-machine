package entities

import (
	"strings"
)

type Ingredient struct {
	ID       string
	Quantity int
}

type Item struct {
	ID          string
	Ingredients []Ingredient
}

type RejectReason struct {
	RejectReasonMsg string
}

func (r RejectReason) String() string {
	return r.RejectReasonMsg
}

type GetItemOutcome string

var (
	GetItemOutcomePrepared    GetItemOutcome = "PREPARED"
	GetItemOutcomeNotPrepared GetItemOutcome = "NOT_PREPARED"
)

type GetItemResponse struct {
	Item          Item
	Outcome       GetItemOutcome
	RejectReasons []RejectReason
}

func (g GetItemResponse) String() string {
	resp := strings.Join([]string{g.Item.ID, string(g.Outcome), " "}, " : ")
	if g.Outcome == GetItemOutcomeNotPrepared {
		for _, reason := range g.RejectReasons {
			resp = resp + reason.String()
		}
	}
	resp += "\n"
	return resp
}

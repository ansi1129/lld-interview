package main

import (
	"fmt"
	"math"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

//////////////////////////////////////////////////////
// MODELS
//////////////////////////////////////////////////////

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type Group struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	Members map[string]*User `json:"members"`
}

type Split struct {
	UserID string  `json:"user_id"`
	Amount float64 `json:"amount"`
}

type ExpenseType string

const (
	EQUAL ExpenseType = "EQUAL"
	EXACT ExpenseType = "EXACT"
)

type Expense struct {
	ID      string      `json:"id"`
	GroupID string      `json:"group_id"`
	PaidBy  string      `json:"paid_by"`
	Amount  float64     `json:"amount"`
	Type    ExpenseType `json:"type"`
	Splits  []Split     `json:"splits"`
}

type Settlement struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Amount float64 `json:"amount"`
}

//////////////////////////////////////////////////////
// GLOBAL STORAGE (IN-MEMORY)
//////////////////////////////////////////////////////

var users = make(map[string]*User)
var groups = make(map[string]*Group)

//////////////////////////////////////////////////////
// BALANCE SERVICE
//////////////////////////////////////////////////////

type BalanceService struct {
	balance map[string]map[string]map[string]float64
	mutex   sync.RWMutex
}

func NewBalanceService() *BalanceService {
	return &BalanceService{
		balance: make(map[string]map[string]map[string]float64),
	}
}

var bs = NewBalanceService()

func (b *BalanceService) update(groupID, payer string, splits []Split) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.balance[groupID] == nil {
		b.balance[groupID] = make(map[string]map[string]float64)
	}

	for _, s := range splits {
		if s.UserID == payer {
			continue
		}

		if b.balance[groupID][payer] == nil {
			b.balance[groupID][payer] = make(map[string]float64)
		}
		if b.balance[groupID][s.UserID] == nil {
			b.balance[groupID][s.UserID] = make(map[string]float64)
		}

		b.balance[groupID][payer][s.UserID] += s.Amount
		b.balance[groupID][s.UserID][payer] -= s.Amount
	}
}

//////////////////////////////////////////////////////
// STRATEGY
//////////////////////////////////////////////////////

type SplitStrategy interface {
	Calculate(*Expense) ([]Split, error)
}

type EqualSplit struct{}

func (e *EqualSplit) Calculate(exp *Expense) ([]Split, error) {
	n := len(exp.Splits)
	base := exp.Amount / float64(n)

	var res []Split
	for _, s := range exp.Splits {
		res = append(res, Split{s.UserID, base})
	}
	return res, nil
}

type ExactSplit struct{}

func (e *ExactSplit) Calculate(exp *Expense) ([]Split, error) {
	sum := 0.0
	for _, s := range exp.Splits {
		sum += s.Amount
	}
	if sum != exp.Amount {
		return nil, fmt.Errorf("invalid split")
	}
	return exp.Splits, nil
}

//////////////////////////////////////////////////////
// EXPENSE SERVICE
//////////////////////////////////////////////////////

var strategies = map[ExpenseType]SplitStrategy{
	EQUAL: &EqualSplit{},
	EXACT: &ExactSplit{},
}

//////////////////////////////////////////////////////
// GREEDY SETTLEMENT
//////////////////////////////////////////////////////

func simplify(groupID string) []Settlement {
	net := make(map[string]float64)

	for u1, mp := range bs.balance[groupID] {
		for u2, amt := range mp {
			if amt > 0 {
				net[u1] += amt
				net[u2] -= amt
			}
		}
	}

	type Node struct {
		user string
		amt  float64
	}

	var creditors, debtors []Node

	for u, amt := range net {
		if amt > 0 {
			creditors = append(creditors, Node{u, amt})
		} else if amt < 0 {
			debtors = append(debtors, Node{u, -amt})
		}
	}

	i, j := 0, 0
	var res []Settlement

	for i < len(debtors) && j < len(creditors) {
		minAmt := math.Min(debtors[i].amt, creditors[j].amt)

		res = append(res, Settlement{
			From:   debtors[i].user,
			To:     creditors[j].user,
			Amount: minAmt,
		})

		debtors[i].amt -= minAmt
		creditors[j].amt -= minAmt

		if debtors[i].amt == 0 {
			i++
		}
		if creditors[j].amt == 0 {
			j++
		}
	}

	return res
}

//////////////////////////////////////////////////////
// HANDLERS
//////////////////////////////////////////////////////

func createUser(c *gin.Context) {
	var u User
	if err := c.BindJSON(&u); err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}
	users[u.ID] = &u
	c.JSON(http.StatusOK, u)
}

func createGroup(c *gin.Context) {
	var g Group
	if err := c.BindJSON(&g); err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}
	g.Members = make(map[string]*User)
	groups[g.ID] = &g
	c.JSON(http.StatusOK, g)
}

func addUserToGroup(c *gin.Context) {
	groupID := c.Param("id")

	var body struct {
		UserID string `json:"user_id"`
	}
	c.BindJSON(&body)

	group := groups[groupID]
	user := users[body.UserID]

	group.Members[user.ID] = user

	c.JSON(http.StatusOK, group)
}

func addExpense(c *gin.Context) {
	var exp Expense
	if err := c.BindJSON(&exp); err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}

	strategy := strategies[exp.Type]
	splits, err := strategy.Calculate(&exp)
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	bs.update(exp.GroupID, exp.PaidBy, splits)

	c.JSON(http.StatusOK, "expense added")
}

func getBalances(c *gin.Context) {
	groupID := c.Param("id")

	var result []string

	for u1, mp := range bs.balance[groupID] {
		for u2, amt := range mp {
			if amt > 0 {
				result = append(result,
					fmt.Sprintf("%s owes %s: %.2f", u2, u1, amt))
			}
		}
	}

	c.JSON(http.StatusOK, result)
}

func getSettlements(c *gin.Context) {
	groupID := c.Param("id")
	res := simplify(groupID)
	c.JSON(http.StatusOK, res)
}

//////////////////////////////////////////////////////
// MAIN
//////////////////////////////////////////////////////

func main() {

	r := gin.Default()

	r.POST("/users", createUser)
	r.POST("/groups", createGroup)
	r.POST("/groups/:id/users", addUserToGroup)
	r.POST("/expenses", addExpense)
	r.GET("/groups/:id/balances", getBalances)
	r.GET("/groups/:id/settlements", getSettlements)

	r.Run(":8080")
}

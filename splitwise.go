package main

import (
	"fmt"
	"math"
	"sync"
)

//////////////////////////////////////////////////////
// ENUMS
//////////////////////////////////////////////////////

type ExpenseType string

const (
	EQUAL ExpenseType = "EQUAL"
	EXACT ExpenseType = "EXACT"
)

//////////////////////////////////////////////////////
// MODELS
//////////////////////////////////////////////////////

type User struct {
	ID string
}

type Group struct {
	ID      string
	Members map[string]*User
}

type Split struct {
	UserID string
	Amount float64
}

type Expense struct {
	GroupID string
	PaidBy  string
	Amount  float64
	Type    ExpenseType
	Splits  []Split
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
	if n == 0 {
		return nil, fmt.Errorf("no users")
	}

	base := math.Floor((exp.Amount/float64(n))*100) / 100
	total := base * float64(n)
	rem := math.Round((exp.Amount-total)*100) / 100

	res := []Split{}
	for i, s := range exp.Splits {
		amt := base
		if i == 0 {
			amt += rem
		}
		res = append(res, Split{s.UserID, amt})
	}
	return res, nil
}

type ExactSplit struct{}

func (e *ExactSplit) Calculate(exp *Expense) ([]Split, error) {
	sum := 0.0
	for _, s := range exp.Splits {
		sum += s.Amount
	}
	if math.Round(sum*100)/100 != exp.Amount {
		return nil, fmt.Errorf("invalid exact split")
	}
	return exp.Splits, nil
}

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

func (b *BalanceService) showGroup(groupID string) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	found := false

	for u1, mp := range b.balance[groupID] {
		for u2, amt := range mp {
			if amt > 0 {
				fmt.Printf("%s owes %s: %.2f\n", u2, u1, amt)
				found = true
			}
		}
	}

	if !found {
		fmt.Println("No balances")
	}
}

//////////////////////////////////////////////////////
// EXPENSE SERVICE
//////////////////////////////////////////////////////

type ExpenseService struct {
	balanceService *BalanceService
	groups         map[string]*Group
	strategies     map[ExpenseType]SplitStrategy
}

func NewExpenseService(b *BalanceService) *ExpenseService {
	return &ExpenseService{
		balanceService: b,
		groups:         make(map[string]*Group),
		strategies: map[ExpenseType]SplitStrategy{
			EQUAL: &EqualSplit{},
			EXACT: &ExactSplit{},
		},
	}
}

func (e *ExpenseService) AddExpense(exp *Expense) error {

	group := e.groups[exp.GroupID]
	if group == nil {
		return fmt.Errorf("group not found")
	}

	// Validate users
	for _, s := range exp.Splits {
		if group.Members[s.UserID] == nil {
			return fmt.Errorf("user %s not in group", s.UserID)
		}
	}

	// Strategy
	strategy := e.strategies[exp.Type]
	splits, err := strategy.Calculate(exp)
	if err != nil {
		return err
	}

	// Update balance
	e.balanceService.update(exp.GroupID, exp.PaidBy, splits)

	return nil
}

//////////////////////////////////////////////////////
// GROUP SERVICE
//////////////////////////////////////////////////////

type GroupService struct {
	groups map[string]*Group
}

func NewGroupService() *GroupService {
	return &GroupService{
		groups: make(map[string]*Group),
	}
}

func (g *GroupService) CreateGroup(id string) *Group {
	group := &Group{
		ID:      id,
		Members: make(map[string]*User),
	}
	g.groups[id] = group
	return group
}

func (g *GroupService) AddUser(groupID string, user *User) {
	group := g.groups[groupID]
	group.Members[user.ID] = user
}

//////////////////////////////////////////////////////
// MAIN
//////////////////////////////////////////////////////

func main() {

	bs := NewBalanceService()
	es := NewExpenseService(bs)
	gs := NewGroupService()

	// Create group
	group := gs.CreateGroup("g1")
	fmt.Println("Created group:", group.ID)

	u1 := &User{"u1"}
	u2 := &User{"u2"}
	u3 := &User{"u3"}

	gs.AddUser("g1", u1)
	gs.AddUser("g1", u2)
	gs.AddUser("g1", u3)

	// Link group to expense service
	es.groups = gs.groups

	// Expense 1: EQUAL
	exp1 := &Expense{
		GroupID: "g1",
		PaidBy:  "u1",
		Amount:  900,
		Type:    EQUAL,
		Splits: []Split{
			{UserID: "u1"},
			{UserID: "u2"},
			{UserID: "u3"},
		},
	}

	es.AddExpense(exp1)

	// Expense 2: EXACT
	exp2 := &Expense{
		GroupID: "g1",
		PaidBy:  "u1",
		Amount:  500,
		Type:    EXACT,
		Splits: []Split{
			{UserID: "u2", Amount: 200},
			{UserID: "u3", Amount: 300},
		},
	}

	es.AddExpense(exp2)

	fmt.Println("\nGroup Balances:")
	bs.showGroup("g1")
}

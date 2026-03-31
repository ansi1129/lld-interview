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
	ID       string
	Username string
}

type Group struct {
	ID      string
	Name    string
	Members map[string]*User
}

type Split struct {
	UserID string
	Amount float64
}

type Expense struct {
	ID      string
	GroupID string
	PaidBy  string
	Amount  float64
	Type    ExpenseType
	Splits  []Split
}

type Settlement struct {
	From   string
	To     string
	Amount float64
}

//////////////////////////////////////////////////////
// STRATEGY PATTERN
//////////////////////////////////////////////////////

type SplitStrategy interface {
	Calculate(*Expense) ([]Split, error)
}

// Equal Split
type EqualSplit struct{}

func (e *EqualSplit) Calculate(exp *Expense) ([]Split, error) {
	n := len(exp.Splits)
	if n == 0 {
		return nil, fmt.Errorf("no users")
	}

	base := math.Floor((exp.Amount/float64(n))*100) / 100
	total := base * float64(n)
	rem := math.Round((exp.Amount-total)*100) / 100

	var res []Split
	for i, s := range exp.Splits {
		amt := base
		if i == 0 {
			amt += rem
		}
		res = append(res, Split{s.UserID, amt})
	}
	return res, nil
}

// Exact Split
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

func (b *BalanceService) showGroup(groupID string, users map[string]*User) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	found := false

	for u1, mp := range b.balance[groupID] {
		for u2, amt := range mp {
			if amt > 0 {
				fmt.Printf("%s owes %s: %.2f\n",
					users[u2].Username,
					users[u1].Username,
					amt,
				)
				found = true
			}
		}
	}

	if !found {
		fmt.Println("No balances")
	}
}

//////////////////////////////////////////////////////
// GREEDY SETTLEMENT
//////////////////////////////////////////////////////

func (b *BalanceService) Simplify(groupID string) []Settlement {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	net := make(map[string]float64)

	for u1, mp := range b.balance[groupID] {
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
	var result []Settlement

	for i < len(debtors) && j < len(creditors) {
		minAmt := math.Min(debtors[i].amt, creditors[j].amt)

		result = append(result, Settlement{
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

	return result
}

func (b *BalanceService) ShowSimplified(groupID string, users map[string]*User) {
	settlements := b.Simplify(groupID)

	if len(settlements) == 0 {
		fmt.Println("No settlements needed")
		return
	}

	for _, s := range settlements {
		fmt.Printf("%s pays %s: %.2f\n",
			users[s.From].Username,
			users[s.To].Username,
			s.Amount,
		)
	}
}

//////////////////////////////////////////////////////
// SERVICES
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

	strategy := e.strategies[exp.Type]
	splits, err := strategy.Calculate(exp)
	if err != nil {
		return err
	}

	e.balanceService.update(exp.GroupID, exp.PaidBy, splits)
	return nil
}

type GroupService struct {
	groups map[string]*Group
}

func NewGroupService() *GroupService {
	return &GroupService{
		groups: make(map[string]*Group),
	}
}

func (g *GroupService) CreateGroup(id, name string) *Group {
	group := &Group{
		ID:      id,
		Name:    name,
		Members: make(map[string]*User),
	}
	g.groups[id] = group
	return group
}

func (g *GroupService) AddUser(groupID string, user *User) {
	g.groups[groupID].Members[user.ID] = user
}

//////////////////////////////////////////////////////
// TEST RUNNERS
//////////////////////////////////////////////////////

func printAll(bs *BalanceService, group *Group) {
	fmt.Println("\n--- Raw Balances ---")
	bs.showGroup(group.ID, group.Members)

	fmt.Println("\n--- Simplified Settlements ---")
	bs.ShowSimplified(group.ID, group.Members)
}

func runBasicCase() {
	fmt.Println("\n========== BASIC ==========")

	bs := NewBalanceService()
	es := NewExpenseService(bs)
	gs := NewGroupService()

	group := gs.CreateGroup("g1", "Trip")

	u1 := &User{"u1", "Ankit"}
	u2 := &User{"u2", "Rahul"}
	u3 := &User{"u3", "Priya"}

	gs.AddUser("g1", u1)
	gs.AddUser("g1", u2)
	gs.AddUser("g1", u3)

	es.groups = gs.groups

	es.AddExpense(&Expense{
		ID:      "e1",
		GroupID: "g1",
		PaidBy:  "u1",
		Amount:  900,
		Type:    EQUAL,
		Splits: []Split{
			{"u1", 0}, {"u2", 0}, {"u3", 0},
		},
	})

	printAll(bs, group)
}

func runCircularCase() {
	fmt.Println("\n========== CIRCULAR ==========")

	bs := NewBalanceService()
	es := NewExpenseService(bs)
	gs := NewGroupService()

	group := gs.CreateGroup("g2", "Circle")

	u1 := &User{"u1", "A"}
	u2 := &User{"u2", "B"}
	u3 := &User{"u3", "C"}

	gs.AddUser("g2", u1)
	gs.AddUser("g2", u2)
	gs.AddUser("g2", u3)

	es.groups = gs.groups

	es.AddExpense(&Expense{"e1", "g2", "u1", 100, EXACT, []Split{{"u2", 100}}})
	es.AddExpense(&Expense{"e2", "g2", "u2", 100, EXACT, []Split{{"u3", 100}}})
	es.AddExpense(&Expense{"e3", "g2", "u3", 100, EXACT, []Split{{"u1", 100}}})

	printAll(bs, group)
}

//////////////////////////////////////////////////////
// MAIN
//////////////////////////////////////////////////////

func main() {
	runBasicCase()
	runCircularCase()
}

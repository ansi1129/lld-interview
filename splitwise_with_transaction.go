package main

import (
	"fmt"
	"math"
	"sync"
)

type ExpenseType string

const (
	EQUAL ExpenseType = "EQUAL"
	EXACT ExpenseType = "EXACT"
)

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

type Transaction struct {
	GroupID  string
	FromUser string
	ToUser   string
	Amount   float64
	Type     string
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
		return nil, fmt.Errorf("no users to split")
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

type ExactSplit struct{}

func (e *ExactSplit) Calculate(exp *Expense) ([]Split, error) {
	sum := 0.0
	for _, s := range exp.Splits {
		sum += s.Amount
	}
	if math.Round(sum*100)/100 != exp.Amount {
		return nil, fmt.Errorf("invalid split")
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

	groupBalances, ok := b.balance[groupID]
	if !ok {
		fmt.Println("No balances")
		return
	}

	found := false

	for u1, mp := range groupBalances {
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
// TRANSACTION SERVICE
//////////////////////////////////////////////////////

type TransactionService struct {
	transactions []Transaction
	mutex        sync.Mutex
}

func NewTransactionService() *TransactionService {
	return &TransactionService{}
}

func (t *TransactionService) add(tx Transaction) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.transactions = append(t.transactions, tx)
}

//////////////////////////////////////////////////////
// EXPENSE SERVICE
//////////////////////////////////////////////////////

type ExpenseService struct {
	balanceService     *BalanceService
	transactionService *TransactionService
	groups             map[string]*Group
	strategies         map[ExpenseType]SplitStrategy
}

func NewExpenseService(b *BalanceService, t *TransactionService) *ExpenseService {
	return &ExpenseService{
		balanceService:     b,
		transactionService: t,
		groups:             make(map[string]*Group),
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

	for _, s := range exp.Splits {
		if group.Members[s.UserID] == nil {
			return fmt.Errorf("user not in group")
		}
	}

	strategy := e.strategies[exp.Type]
	splits, err := strategy.Calculate(exp)
	if err != nil {
		return err
	}

	for _, s := range splits {
		if s.UserID == exp.PaidBy {
			continue
		}

		e.transactionService.add(Transaction{
			GroupID:  exp.GroupID,
			FromUser: s.UserID,
			ToUser:   exp.PaidBy,
			Amount:   s.Amount,
			Type:     "EXPENSE",
		})
	}

	e.balanceService.update(exp.GroupID, exp.PaidBy, splits)

	return nil
}

//////////////////////////////////////////////////////
// MAIN
//////////////////////////////////////////////////////

func main() {

	bs := NewBalanceService()
	ts := NewTransactionService()
	es := NewExpenseService(bs, ts)

	group := &Group{
		ID: "g1",
		Members: map[string]*User{
			"u1": &User{ID: "u1"},
			"u2": &User{ID: "u2"},
			"u3": &User{ID: "u3"},
		},
	}

	es.groups["g1"] = group

	exp := &Expense{
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

	err := es.AddExpense(exp)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("\nBalances:")
	bs.showGroup("g1")
}

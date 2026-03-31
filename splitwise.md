# 💸 Splitwise — Low Level Design (LLD)

> A production-ready expense-sharing system supporting group expenses, multiple split strategies, and greedy debt simplification to minimize transactions.

---

## 📋 Table of Contents

- [Problem Understanding](#-problem-understanding)
- [Requirements](#-requirements--clarifications)
- [High-Level Design](#-high-level-design)
- [Core Design Decisions](#-core-design-decisions)
- [Models](#-models)
- [Algorithms](#-algorithms-core-logic)
- [Implementation](#-implementation-interview-ready)
- [Complexity](#-complexity)
- [Design Patterns](#-design-patterns-used)
- [SOLID Principles](#-solid-principles)
- [Production Enhancements](#-production-enhancements)
- [Conclusion](#-final-conclusion)

---

## 🎯 Problem Understanding

> *"We need to design a Splitwise-like system where users can share expenses in groups, split them in different ways, and efficiently settle debts."*

---

## 📌 Requirements & Clarifications

### ✅ Functional
- Create users
- Create groups & add users to a group
- Add expenses with split types:
  - **Equal Split** — divide equally among members
  - **Exact Split** — user-specified amounts
- Show balances between users
- Simplify debts (minimum number of transactions)

### ⚠️ Non-Functional
- Low latency
- Thread-safe balance updates
- Extensible for new split types
- Consistent balances across operations
- Scalable for large groups

---

## 🏗️ High-Level Design

```
User
  ↓
Group Service
  ↓
Expense Service
  ↓
Split Strategy (Equal / Exact)
  ↓
Balance Service (Ledger)
  ↓
Greedy Settlement Engine
```

---

## 🧠 Core Design Decisions

### 1. Strategy Pattern
- Each split type encapsulates its own logic
- New split types can be added without modifying existing code
- Clean, extensible, and testable

### 2. Ledger-Based Balance Tracking
```
balance[A][B] = amount  →  B owes A
```
- Pairwise ledger maintained per group
- Fast O(1) lookup per user pair

### 3. Net Balance for Debt Optimization
- Convert pairwise balances → net balance per user
- Enables greedy settlement to minimize transaction count

---

## 🏗️ Models

```go
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
    Type    string
    Splits  []Split
}
```

---

## 🧠 Algorithms (Core Logic)

### 🔹 Equal Split
```
1. Divide total amount by number of users
2. Handle floating-point rounding
3. Assign remainder to one user (e.g., payer)
```

### 🔹 Exact Split
```
1. Accept user-provided amounts per member
2. Validate: sum of splits == total expense amount
3. Reject if validation fails
```

### 🔹 Balance Update (Core Ledger Logic)
```
For each split:
  1. Skip the payer
  2. Others owe the payer their share

  balance[payer][user] += amount
  balance[user][payer] -= amount
```

### 🔹 Greedy Debt Simplification *(SDE-3 Level)*
```
1. Compute net balance for each user
   (sum of all amounts owed to them minus owed by them)

2. Separate users into:
   - Creditors (net balance > 0)
   - Debtors   (net balance < 0)

3. Greedily match debtors to creditors:
   - Settle min(debt, credit) at each step
   - Update both parties
   - Remove settled users

4. Result: minimum number of transactions to clear all debts
```

---

## 💻 Implementation (Interview Ready)

### Split Strategy Interface
```go
type SplitStrategy interface {
    Calculate(*Expense) ([]Split, error)
}
```

### Equal Split
```go
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
```

### Exact Split
```go
type ExactSplit struct{}

func (e *ExactSplit) Calculate(exp *Expense) ([]Split, error) {
    sum := 0.0
    for _, s := range exp.Splits {
        sum += s.Amount
    }
    if sum != exp.Amount {
        return nil, fmt.Errorf("invalid split: amounts don't add up")
    }
    return exp.Splits, nil
}
```

### Balance Update
```go
func (b *BalanceService) update(groupID, payer string, splits []Split) {
    for _, s := range splits {
        if s.UserID == payer {
            continue
        }
        b.balance[payer][s.UserID] += s.Amount
        b.balance[s.UserID][payer] -= s.Amount
    }
}
```

### Greedy Settlement (Core Idea)
```go
// Pseudocode
for len(debtors) > 0 && len(creditors) > 0 {
    min := math.Min(debt, credit)
    // debtor pays creditor `min` amount
    // update both balances
    // remove if fully settled
}
```

---

## 🔍 Component Breakdown

| Component | Responsibility |
|---|---|
| `BalanceService` | Maintains pairwise ledger of debts, thread-safe |
| `SplitStrategy` | Clean separation of split logic, easily extendable |
| `GreedySettlement` | Reduces complex debt graph to minimum transactions |

---

## ⏱️ Complexity

| Operation | Time Complexity |
|---|---|
| `AddExpense` | O(n) |
| `ShowBalance` | O(n²) |
| `SimplifyDebts` | O(n log n) |
| Space | O(n²) — pairwise ledger |

---

## 🎯 Design Patterns Used

| Pattern | Usage |
|---|---|
| **Strategy** | Pluggable split algorithms (Equal, Exact, Percentage…) |
| **Factory** *(optional)* | Create correct strategy based on expense type |

---

## ✅ SOLID Principles

| Principle | Application |
|---|---|
| **SRP** | Each service has a single, well-defined responsibility |
| **OCP** | New split types added without modifying `ExpenseService` |
| **DIP** | `ExpenseService` depends on `SplitStrategy` interface, not concrete types |
| **LSP** | All split strategies are interchangeable via the interface |

---

## 🚀 Production Enhancements

- 🗄️ **MySQL** for expense & user persistence
- ⚡ **Redis** for fast balance lookup and caching
- 📜 **Transaction history** per group/user
- 🔁 **Idempotency keys** to prevent duplicate expense entries
- 📊 **Percentage split** support as a new strategy
- 🔒 **Distributed locking** for concurrent balance updates

---

## 📁 Project Structure

```
splitwise/
├── models/
│   ├── user.go
│   ├── group.go
│   └── expense.go
├── strategies/
│   ├── split_strategy.go          # Interface
│   ├── equal_split.go
│   └── exact_split.go
├── services/
│   ├── group_service.go
│   ├── expense_service.go
│   └── balance_service.go
├── settlement/
│   └── greedy_settlement.go       # Debt simplification
├── factory/
│   └── split_factory.go
└── main.go
```

---

## 🏁 Final Conclusion

> *"I designed the system using a strategy pattern for different split types and a ledger-based balance system. Additionally, I implemented a greedy algorithm to simplify debts by converting pairwise balances into net balances and minimizing the number of transactions."*

---

## 🔥 Key Insight

> **While the ledger tracks pairwise debts, the greedy settlement ensures optimal transaction reduction — which is critical for user experience in real-world systems like Splitwise.**
# 🅿️ Parking Lot System Design

## 1. Problem Statement

> Design a parking lot system that can park and unpark vehicles, assign appropriate slots, and calculate parking fees.

---

## 2. Requirements & Clarifications

### ✅ Functional
- Add parking lot
- Add parking floors
- Add parking slots
- Park vehicle
- Unpark vehicle
- Calculate parking fee
- Get available slots
- Support multiple vehicle types: **Bike**, **Car**, **Truck**

### ❗ Non-Functional
- Low latency (real-time parking)
- Thread-safe
- Scalable (multiple floors)
- Extensible (new vehicle types)
- High availability

---

## 3. High-Level Design

```
Vehicle → ParkingSlot → ParkingFloor → ParkingLot
                     ↓
               Ticket Service
                     ↓
               Payment Service
```

---

## 4. Core Design Decisions

| Decision | Approach |
|---|---|
| **Pricing** | Strategy Pattern — different rates per vehicle type |
| **Slot Allocation** | First available slot (extendable to nearest/priority) |
| **Tracking** | Ticket-based system — park generates a ticket, unpark uses it |

---

## 5. Models

```go
type VehicleType string

const (
    BIKE  VehicleType = "BIKE"
    CAR   VehicleType = "CAR"
    TRUCK VehicleType = "TRUCK"
)

type Vehicle struct {
    Number string
    Type   VehicleType
}

type ParkingSlot struct {
    ID         string
    Type       VehicleType
    IsOccupied bool
    Vehicle    *Vehicle
}

type ParkingFloor struct {
    ID    string
    Slots map[string]*ParkingSlot
}

type ParkingLot struct {
    ID     string
    Floors map[string]*ParkingFloor
}

type Ticket struct {
    ID        string
    Vehicle   *Vehicle
    SlotID    string
    FloorID   string
    EntryTime int64
}
```

---

## 6. Algorithms

### 🔍 Slot Allocation
> "Find the first available slot matching vehicle type."

1. Iterate floors
2. Iterate slots on each floor
3. Match type + availability
4. Assign slot

### 🚗 Parking Flow
1. Find matching slot
2. Mark slot as occupied
3. Create and return ticket

### 🚪 Unparking Flow
1. Retrieve ticket by ID
2. Free the slot
3. Calculate fee based on duration

### 💰 Fee Calculation
```
fee = hours × rate(vehicleType)
```

---

## 7. Complete Implementation

```go
package main

import (
    "fmt"
    "sync"
    "time"
)

// ── MODELS ──────────────────────────────────────────────────────────────────

type VehicleType string

const (
    BIKE  VehicleType = "BIKE"
    CAR   VehicleType = "CAR"
    TRUCK VehicleType = "TRUCK"
)

type Vehicle struct {
    Number string
    Type   VehicleType
}

type ParkingSlot struct {
    ID         string
    Type       VehicleType
    IsOccupied bool
    Vehicle    *Vehicle
}

type ParkingFloor struct {
    ID    string
    Slots map[string]*ParkingSlot
}

type ParkingLot struct {
    ID     string
    Floors map[string]*ParkingFloor
    mutex  sync.Mutex
}

type Ticket struct {
    ID        string
    Vehicle   *Vehicle
    SlotID    string
    FloorID   string
    EntryTime time.Time
}

// ── PARKING SERVICE ──────────────────────────────────────────────────────────

type ParkingService struct {
    lot     *ParkingLot
    tickets map[string]*Ticket
}

func NewParkingService(lot *ParkingLot) *ParkingService {
    return &ParkingService{
        lot:     lot,
        tickets: make(map[string]*Ticket),
    }
}

// ── PARK VEHICLE ─────────────────────────────────────────────────────────────

func (p *ParkingService) Park(vehicle *Vehicle) (*Ticket, error) {
    p.lot.mutex.Lock()
    defer p.lot.mutex.Unlock()

    for _, floor := range p.lot.Floors {
        for _, slot := range floor.Slots {
            if !slot.IsOccupied && slot.Type == vehicle.Type {
                slot.IsOccupied = true
                slot.Vehicle = vehicle

                ticket := &Ticket{
                    ID:        fmt.Sprintf("t-%d", time.Now().UnixNano()),
                    Vehicle:   vehicle,
                    SlotID:    slot.ID,
                    FloorID:   floor.ID,
                    EntryTime: time.Now(),
                }

                p.tickets[ticket.ID] = ticket
                return ticket, nil
            }
        }
    }

    return nil, fmt.Errorf("no slot available")
}

// ── UNPARK VEHICLE ────────────────────────────────────────────────────────────

func (p *ParkingService) Unpark(ticketID string) (float64, error) {
    p.lot.mutex.Lock()
    defer p.lot.mutex.Unlock()

    ticket := p.tickets[ticketID]
    if ticket == nil {
        return 0, fmt.Errorf("invalid ticket")
    }

    floor := p.lot.Floors[ticket.FloorID]
    slot := floor.Slots[ticket.SlotID]

    slot.IsOccupied = false
    slot.Vehicle = nil

    duration := time.Since(ticket.EntryTime).Hours()
    rate := getRate(ticket.Vehicle.Type)
    fee := duration * rate

    delete(p.tickets, ticketID)
    return fee, nil
}

// ── PRICING STRATEGY ──────────────────────────────────────────────────────────

func getRate(vType VehicleType) float64 {
    switch vType {
    case BIKE:
        return 10
    case CAR:
        return 20
    case TRUCK:
        return 50
    }
    return 0
}

// ── MAIN ──────────────────────────────────────────────────────────────────────

func main() {
    lot := &ParkingLot{
        ID:     "P1",
        Floors: make(map[string]*ParkingFloor),
    }

    floor := &ParkingFloor{
        ID:    "F1",
        Slots: make(map[string]*ParkingSlot),
    }

    floor.Slots["S1"] = &ParkingSlot{"S1", CAR, false, nil}
    floor.Slots["S2"] = &ParkingSlot{"S2", BIKE, false, nil}
    lot.Floors["F1"] = floor

    service := NewParkingService(lot)

    car := &Vehicle{"KA01", CAR}
    ticket, _ := service.Park(car)
    fmt.Println("Parked:", ticket.ID)

    time.Sleep(2 * time.Second)

    fee, _ := service.Unpark(ticket.ID)
    fmt.Println("Fee:", fee)
}
```

---

## 8. Code Explanation

| Component | Role |
|---|---|
| `ParkingService` | Core business logic — handles park/unpark |
| `Ticket` | Tracks entry time, used for billing |
| Slot Allocation | Linear scan; extensible to priority/nearest strategies |
| `mutex` | Ensures thread safety on concurrent park/unpark |

---

## 9. Complexity

| Operation | Time Complexity | Space Complexity |
|---|---|---|
| Park | O(n slots) | O(n slots) |
| Unpark | O(1) | — |

---

## 10. Design Principles

### SOLID
- **SRP** — Separate services for parking, ticketing, and pricing
- **OCP** — New vehicle types added without modifying existing logic
- **DIP** — Pricing abstracted via `getRate()` strategy function

### Design Patterns
- **Strategy Pattern** — Pluggable pricing per vehicle type
- **Singleton** — Single parking lot instance

---

## 11. Production Enhancements

- **Database** — Persist tickets and slot state (PostgreSQL/MySQL)
- **Redis** — Cache slot availability for low-latency lookups
- **Slot Reservation** — Pre-book slots before arrival
- **Nearest Slot Allocation** — Priority queue for optimal slot selection
- **Multi-Entry Gates** — Distributed ticket issuance at each gate
- **Payment Gateway** — Integrate Stripe/Razorpay for digital payments

---

## 12. Interview Summary

> "I designed the parking lot using object-oriented principles with clear separation between slot management, ticketing, and pricing. The system allocates slots efficiently and calculates fees using a strategy-based pricing model."

### 🔥 Killer Add-on

> "We can optimize slot lookup using indexed maps or priority queues to achieve near **O(1)** allocation instead of O(n) linear scan."
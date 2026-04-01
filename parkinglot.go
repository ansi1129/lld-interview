package main

import (
	"fmt"
	"sync"
	"time"
)

//////////////////////////////////////////////////////
// MODELS
//////////////////////////////////////////////////////

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

//////////////////////////////////////////////////////
// PARKING SERVICE
//////////////////////////////////////////////////////

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

//////////////////////////////////////////////////////
// PARK VEHICLE
//////////////////////////////////////////////////////

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

//////////////////////////////////////////////////////
// UNPARK VEHICLE
//////////////////////////////////////////////////////

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

//////////////////////////////////////////////////////
// PRICING STRATEGY
//////////////////////////////////////////////////////

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

//////////////////////////////////////////////////////
// MAIN
//////////////////////////////////////////////////////

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

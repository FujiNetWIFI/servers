package main

import (
	"errors"
	"fmt"
)

type PlayerCred struct {
	Name string // IsPrintableAscii with no spaces
}

// TODO: to test

func (p *PlayerCred) CheckInput() (err error) {
	return errors.Join(
		ErrorIf(!IsPrintableAscii(p.Name), fmt.Errorf("key: 'PlayerCred.Name' Error: Value must contain printable characters only with no spaces")),
	)
}

func (p *PlayerCred) IsCorrect() bool {
	return nil == p.CheckInput()
}

type Shoot struct {
	Name  string // IsPrintableAscii
	Angle int    // ∋ [-180, 180]
	Power int    // ∋ [0, 100]
}

func (s *Shoot) CheckInput() (err error) {
	return errors.Join(
		ErrorIf(!IsPrintableAscii(s.Name), fmt.Errorf("key: 'Shoot.Name' Error: Value must contain printable characters only")),
		ErrorIf(s.Angle < -180 || s.Angle > 180, fmt.Errorf("key: 'Shoot.Angle' Error: Value must be between -180 and 180")),
		ErrorIf(s.Power < 0 || s.Power > 100, fmt.Errorf("key: 'Shoot.Power' Error: Value must be between -180 and 180")),
	)

}

func (s *Shoot) IsCorrect() bool {
	return nil == s.CheckInput()
}

type Checkeable interface {
	CheckInput() (err error)
	IsCorrect() bool
}

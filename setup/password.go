package setup

import (
	"sync"

	sd "github.com/jakebowkett/storydevs"
	"golang.org/x/crypto/bcrypt"
)

func password(c *sd.Config) sd.Password {
	return &pw{cost: bcrypt.MinCost}
}

type pw struct {
	cost   int
	costMu sync.Mutex
}

func (p *pw) SetCost(cost int) {
	p.costMu.Lock()
	p.cost = cost
	p.costMu.Unlock()
}

func (p *pw) Hash(pass string) (hash string, err error) {
	if p.cost < bcrypt.MinCost {
		p.SetCost(bcrypt.MinCost)
	}
	bb, err := bcrypt.GenerateFromPassword([]byte(pass), p.cost)
	if err != nil {
		return hash, err
	}
	return string(bb), err
}

func (p *pw) Compare(pass, hash string) (ok bool, err error) {
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

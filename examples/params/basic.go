package params

import (
	"math/big"
	"time"
)

type Basic struct {
}

func (bs *Basic) Create(userID int64, name, lastName string, expiration time.Duration, amount *big.Int) (int, error) {
	return 0, nil
}

func (bs *Basic) Crazy(data []*time.Time, names map[time.Duration]*big.Int) (a int, b float64, v *big.Int, d string, e Ooops) {
	return
}

func (bs *Basic) List() (a, b, c int) {
	return
}

func (bs *Basic) Sum(a, b, c int) int { return a + b + c }

func (bs *Basic) ignoreIt() {}

package app

import (
	"math/rand"
	"time"
)

// LocTime get local time
func LocTime() time.Time {
	return time.Now().Local()
}

func RandomString(lens int) string {
	rand.Seed(time.Now().UnixNano())
	bytes := make([]byte, lens)
	for i := 0; i < lens; i++ {
		bytes[i] = Pool[rand.Intn(len(Pool))]
	}
	return string(bytes)
}

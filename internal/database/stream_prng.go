package database

import "math/rand"

var Rand *rand.Rand

func init() {
	Rand = rand.New(rand.NewSource(int64(rand.Uint64())))
}

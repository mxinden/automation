package kubernetes

import (
	"math/rand"
	"time"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func init() {
	rand.Seed(time.Now().UnixNano())
}

func getRandomName() string {
	b := make([]rune, 10)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

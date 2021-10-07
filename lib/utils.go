package lib

import (
	"math/rand"
	"time"
)

var Alphabet = []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func init() {
	rand.Seed(int64(time.Now().Nanosecond()))
}

func randomChar() string {
	return string(Alphabet[rand.Intn(len(Alphabet))])
}

func GenRandomString(len int) (link string) {
	for i := 0; i < len; i++ {
		link += randomChar()
	}
	return link
}

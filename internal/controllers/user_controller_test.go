package controllers

import (
	crypto_rand "crypto/rand"
	"log"
	"math/big"
	"testing"
)

type referralCod [6]uint8

func generateReferralCod() referralCod {

	var digMax = big.NewInt(10)
	var c referralCod

	for i := 0; i < 6; i++ {
		d, err := crypto_rand.Int(crypto_rand.Reader, digMax)
		if err != nil {
			panic(err)
		}
		d.Uint64()
		c[i] = uint8(d.Int64())
	}

	return c
}

func TestGenerateCode(t *testing.T) {
	a := generateReferralCod()
	log.Println(a)
}

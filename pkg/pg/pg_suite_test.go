package pg

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

func TestPg(t *testing.T) {
	log = zap.NewNop().Sugar()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pg Suite")
}

// func randomString(length uint) string {
// 	b := make([]byte, length)
// 	for i := range b {
// 		b[i] = charset[seededRand.Intn(len(charset))]
// 	}
// 	return string(b)
// }

// const charset = "abcdefghijklmnopqrstuvwxyz" +
// 	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// var seededRand *rand.Rand = rand.New(
// 	rand.NewSource(time.Now().UnixNano()))

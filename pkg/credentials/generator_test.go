package credentials_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/zhulik/d3/pkg/credentials"
)

var _ = Describe("Credentials Generator", func() {
	Describe("GenerateCredentials", func() {
		It("generates both access key ID and secret access key", func() {
			accessKeyID, secretAccessKey := credentials.GenerateCredentials()
			Expect(accessKeyID).To(HavePrefix("AKIA"))
			Expect(accessKeyID).To(HaveLen(credentials.AccessKeyIDLength))
			Expect(secretAccessKey).To(HaveLen(credentials.SecretAccessKeyLength))
		})

		It("generates unique credentials", func() {
			creds11, creds12 := credentials.GenerateCredentials()

			creds21, creds22 := credentials.GenerateCredentials()

			Expect(creds11).NotTo(Equal(creds21))
			Expect(creds12).NotTo(Equal(creds22))
		})
	})
})

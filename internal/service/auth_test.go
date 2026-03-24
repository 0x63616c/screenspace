package service_test

import (
	"crypto/rand"
	"crypto/rsa"
	"time"

	"github.com/golang-jwt/jwt/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/types"
)

var _ = Describe("AuthService", func() {
	var (
		auth *service.AuthService
		cfg  *config.Config
	)

	BeforeEach(func() {
		cfg = config.DefaultConfig()
		cfg.BcryptCost = 4
		auth = service.NewAuthService(cfg)
	})

	Describe("HashPassword and VerifyPassword", func() {
		It("verifies a correctly hashed password", func() {
			hash, err := auth.HashPassword("secret123")
			Expect(err).NotTo(HaveOccurred())
			Expect(auth.VerifyPassword(hash, "secret123")).To(BeTrue())
		})

		It("returns false for wrong password", func() {
			hash, err := auth.HashPassword("secret123")
			Expect(err).NotTo(HaveOccurred())
			Expect(auth.VerifyPassword(hash, "wrong-password")).To(BeFalse())
		})
	})

	Describe("GenerateToken and ValidateToken", func() {
		It("round-trips with correct claims", func() {
			token, err := auth.GenerateToken("user-123", types.RoleAdmin)
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())

			claims, err := auth.ValidateToken(token)
			Expect(err).NotTo(HaveOccurred())
			Expect(claims.UserID).To(Equal("user-123"))
			Expect(claims.Role).To(Equal(types.RoleAdmin))
		})

		It("fails with expired token", func() {
			cfg.JWTExpiry = -1 * time.Hour
			expiredAuth := service.NewAuthService(cfg)

			token, err := expiredAuth.GenerateToken("user-123", types.RoleUser)
			Expect(err).NotTo(HaveOccurred())

			_, err = expiredAuth.ValidateToken(token)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("parse token"))
		})

		It("fails with wrong secret", func() {
			token, err := auth.GenerateToken("user-123", types.RoleUser)
			Expect(err).NotTo(HaveOccurred())

			otherCfg := config.DefaultConfig()
			otherCfg.JWTSecret = "a-completely-different-secret-that-is-long-enough"
			otherAuth := service.NewAuthService(otherCfg)

			_, err = otherAuth.ValidateToken(token)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("parse token"))
		})

		It("fails with malformed token", func() {
			_, err := auth.ValidateToken("not.a.valid.jwt.token")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("parse token"))
		})

		It("rejects non-HMAC signing method", func() {
			privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			Expect(err).NotTo(HaveOccurred())

			claims := jwt.MapClaims{
				"sub":  "user-123",
				"role": string(types.RoleUser),
				"exp":  time.Now().Add(time.Hour).Unix(),
				"iat":  time.Now().Unix(),
			}
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
			signed, err := token.SignedString(privateKey)
			Expect(err).NotTo(HaveOccurred())

			_, err = auth.ValidateToken(signed)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("parse token"))
		})
	})
})

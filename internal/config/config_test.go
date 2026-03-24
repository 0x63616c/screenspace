package config_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0x63616c/screenspace/server/internal/config"
)

var _ = Describe("Config", func() {
	Describe("Load", func() {
		Context("when required fields are missing", func() {
			BeforeEach(func() {
				GinkgoT().Setenv("DATABASE_URL", "")
				GinkgoT().Setenv("JWT_SECRET", "")
				GinkgoT().Setenv("S3_ENDPOINT", "")
				GinkgoT().Setenv("S3_ACCESS_KEY", "")
				GinkgoT().Setenv("S3_SECRET_KEY", "")
			})

			It("returns an error", func() {
				_, err := config.Load()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when JWT_SECRET is too short", func() {
			BeforeEach(func() {
				GinkgoT().Setenv("DATABASE_URL", "postgres://localhost/test")
				GinkgoT().Setenv("JWT_SECRET", "tooshort")
				GinkgoT().Setenv("S3_ENDPOINT", "http://localhost:9000")
				GinkgoT().Setenv("S3_ACCESS_KEY", "key")
				GinkgoT().Setenv("S3_SECRET_KEY", "secret")
			})

			It("returns an error", func() {
				_, err := config.Load()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with valid required fields and defaults", func() {
			BeforeEach(func() {
				GinkgoT().Setenv("DATABASE_URL", "postgres://localhost/test")
				GinkgoT().Setenv("JWT_SECRET", "a-very-long-secret-that-is-at-least-32-chars!!")
				GinkgoT().Setenv("S3_ENDPOINT", "http://localhost:9000")
				GinkgoT().Setenv("S3_ACCESS_KEY", "key")
				GinkgoT().Setenv("S3_SECRET_KEY", "secret")
			})

			It("loads with default values", func() {
				cfg, err := config.Load()
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.Port).To(Equal("8080"))
				Expect(cfg.DBMaxConns).To(Equal(25))
				Expect(cfg.MaxFileSize).To(Equal(int64(200 * 1024 * 1024)))
				Expect(cfg.UploadRateLimit).To(Equal(5))
			})
		})

		Context("when env vars override defaults", func() {
			BeforeEach(func() {
				GinkgoT().Setenv("DATABASE_URL", "postgres://localhost/test")
				GinkgoT().Setenv("JWT_SECRET", "a-very-long-secret-that-is-at-least-32-chars!!")
				GinkgoT().Setenv("S3_ENDPOINT", "http://localhost:9000")
				GinkgoT().Setenv("S3_ACCESS_KEY", "key")
				GinkgoT().Setenv("S3_SECRET_KEY", "secret")
				GinkgoT().Setenv("PORT", "9090")
				GinkgoT().Setenv("DB_MAX_CONNS", "50")
				GinkgoT().Setenv("UPLOAD_RATE_LIMIT", "10")
			})

			It("uses the overridden values", func() {
				cfg, err := config.Load()
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.Port).To(Equal("9090"))
				Expect(cfg.DBMaxConns).To(Equal(50))
				Expect(cfg.UploadRateLimit).To(Equal(10))
			})
		})
	})
})

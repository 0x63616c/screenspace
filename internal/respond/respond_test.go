package respond_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0x63616c/screenspace/server/internal/respond"
)

var _ = Describe("Respond", func() {
	Describe("JSON", func() {
		It("writes JSON with correct status and content type", func() {
			w := httptest.NewRecorder()
			err := respond.JSON(w, http.StatusOK, map[string]string{"key": "value"})
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("application/json"))
		})
	})

	Describe("Error", func() {
		It("writes a structured error response", func() {
			w := httptest.NewRecorder()
			respond.Error(w, http.StatusNotFound, "not_found", "wallpaper not found")
			Expect(w.Code).To(Equal(http.StatusNotFound))

			var body struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			Expect(json.NewDecoder(w.Body).Decode(&body)).To(Succeed())
			Expect(body.Error.Code).To(Equal("not_found"))
			Expect(body.Error.Message).To(Equal("wallpaper not found"))
		})
	})

	Describe("Paginated", func() {
		It("writes paginated response with items and metadata", func() {
			w := httptest.NewRecorder()
			items := []string{"a", "b", "c"}
			err := respond.Paginated(w, items, 100, 20, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))

			var body struct {
				Items  []string `json:"items"`
				Total  int      `json:"total"`
				Limit  int      `json:"limit"`
				Offset int      `json:"offset"`
			}
			Expect(json.NewDecoder(w.Body).Decode(&body)).To(Succeed())
			Expect(body.Items).To(HaveLen(3))
			Expect(body.Total).To(Equal(100))
			Expect(body.Limit).To(Equal(20))
			Expect(body.Offset).To(Equal(0))
		})
	})

	Describe("ParsePagination", func() {
		DescribeTable("parses pagination params",
			func(query url.Values, wantLimit, wantOffset int) {
				pg := respond.ParsePagination(query, 20, 100)
				Expect(pg.Limit).To(Equal(wantLimit))
				Expect(pg.Offset).To(Equal(wantOffset))
			},
			Entry("defaults", url.Values{}, 20, 0),
			Entry("custom limit", url.Values{"limit": {"50"}}, 50, 0),
			Entry("limit capped at max", url.Values{"limit": {"200"}}, 100, 0),
			Entry("custom offset", url.Values{"offset": {"40"}}, 20, 40),
			Entry("invalid limit uses default", url.Values{"limit": {"abc"}}, 20, 0),
			Entry("negative offset clamped", url.Values{"offset": {"-5"}}, 20, 0),
		)
	})
})

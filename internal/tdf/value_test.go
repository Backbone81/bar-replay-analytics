package tdf_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/tdf"
)

var _ = Describe("Value", func() {
	It("returns its string", func() {
		Expect(tdf.Value("Argent Strata 1.1").String()).To(Equal("Argent Strata 1.1"))
	})

	Context("AsInt", func() {
		DescribeTable("converts",
			func(value tdf.Value, expected int) {
				Expect(value.AsInt()).To(Equal(expected))
			},
			Entry("zero", tdf.Value("0"), 0),
			Entry("a positive integer", tdf.Value("2026"), 2026),
			Entry("a negative integer", tdf.Value("-1"), -1),
			Entry("a leading plus", tdf.Value("+5"), 5),
		)

		DescribeTable("rejects",
			func(value tdf.Value, expectedError string) {
				_, err := value.AsInt()

				Expect(err).To(MatchError(expectedError))
			},
			Entry("an empty value", tdf.Value(""), `"" is not an integer: invalid syntax`),
			Entry("trailing text", tdf.Value("12abc"), `"12abc" is not an integer: invalid syntax`),
			Entry("blanks", tdf.Value(" 1"), `" 1" is not an integer: invalid syntax`),
			Entry("a fraction", tdf.Value("1.5"), `"1.5" is not an integer: invalid syntax`),
			Entry("an overflow", tdf.Value("99999999999999999999"),
				`"99999999999999999999" is not an integer: value out of range`),
		)
	})

	Context("AsFloat", func() {
		DescribeTable("converts",
			func(value tdf.Value, expected float64) {
				Expect(value.AsFloat()).To(Equal(expected))
			},
			Entry("an integer", tdf.Value("1"), 1.0),
			Entry("a float32 as the lobby writes it", tdf.Value("0.74000001"), 0.74000001),
			Entry("a negative number", tdf.Value("-0.5"), -0.5),
			Entry("an exponent", tdf.Value("1e3"), 1000.0),
		)

		DescribeTable("rejects",
			func(value tdf.Value, expectedError string) {
				_, err := value.AsFloat()

				Expect(err).To(MatchError(expectedError))
			},
			Entry("an empty value", tdf.Value(""), `"" is not a number: invalid syntax`),
			Entry("several numbers", tdf.Value("0.99609375 0.546875 0"),
				`"0.99609375 0.546875 0" is not a number: invalid syntax`),
			Entry("an overflow", tdf.Value("1e999"), `"1e999" is not a number: value out of range`),
			Entry("infinity", tdf.Value("inf"), `"inf" is not a finite number`),
			Entry("NaN", tdf.Value("NaN"), `"NaN" is not a finite number`),
		)
	})

	Context("AsBool", func() {
		DescribeTable("converts",
			func(value tdf.Value, expected bool) {
				Expect(value.AsBool()).To(Equal(expected))
			},
			Entry("0", tdf.Value("0"), false),
			Entry("1", tdf.Value("1"), true),
		)

		DescribeTable("rejects",
			func(value tdf.Value) {
				_, err := value.AsBool()

				Expect(err).To(MatchError(`"` + string(value) + `" is not a boolean, want "0" or "1"`))
			},
			Entry("an empty value", tdf.Value("")),
			Entry("true", tdf.Value("true")),
			Entry("2", tdf.Value("2")),
			Entry("01", tdf.Value("01")),
		)
	})
})

package tdf_test

import (
	"encoding/json"
	"maps"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/tdf"
)

var _ = Describe("Section", func() {
	var script *tdf.Section

	BeforeEach(func() {
		var err error
		script, err = tdf.Parse(`
			[game]
			{
				mapname=Argent Strata 1.1;
				[player0] { name=Player; }
				[ai0] { shortname=BARb; [options] { profile=Hard; } }
				[team2] { side=Cortex; }
				[team0] { side=Armada; }
				[team01] {}
				[team+1] {}
				[team] {}
				[teams1] {}
				team3=1;
			}
			[a\b] { c=d; }
		`)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("Value", func() {
		It("returns a value", func() {
			Expect(valueAt(script, `game\mapname`)).To(Equal(tdf.Value("Argent Strata 1.1")))
		})

		It("returns a value in a nested section", func() {
			Expect(valueAt(script, `game\ai0\options\profile`)).To(Equal(tdf.Value("Hard")))
		})

		It("ignores case", func() {
			Expect(valueAt(script, `GAME\Ai0\OPTIONS\Profile`)).To(Equal(tdf.Value("Hard")))
		})

		DescribeTable("returns false for",
			func(location string) {
				value, ok := script.Value(location)

				Expect(ok).To(BeFalse())
				Expect(value).To(BeEmpty())
			},
			Entry("a missing value", `game\missing`),
			Entry("a value in a missing section", `missing\mapname`),
			Entry("a section", `game\player0`),
			Entry("an empty location", ``),
			Entry("an empty name", `game\`),
			Entry("an empty name in between", `game\\mapname`),
			Entry("a name containing the separator", `a\b\c`),
		)

		It("returns false on a nil section", func() {
			var section *tdf.Section

			_, ok := section.Value("a")

			Expect(ok).To(BeFalse())
		})
	})

	Context("Section", func() {
		It("returns a section", func() {
			section := script.Section(`game\ai0`)

			Expect(valueAt(section, "shortname")).To(Equal(tdf.Value("BARb")))
		})

		It("ignores case", func() {
			Expect(script.Section(`GAME\AI0\Options`)).NotTo(BeNil())
		})

		DescribeTable("returns nil for",
			func(location string) {
				Expect(script.Section(location)).To(BeNil())
			},
			Entry("a missing section", `game\missing`),
			Entry("a section in a missing section", `missing\ai0`),
			Entry("a value", `game\mapname`),
			Entry("an empty location", ``),
			Entry("a name containing the separator", `a\b`),
		)

		It("can be chained over missing sections", func() {
			_, ok := script.Section("missing").Section("options").Value("profile")

			Expect(ok).To(BeFalse())
		})
	})

	Context("Values", func() {
		It("returns the values sorted by name, without sections", func() {
			var names []string
			for name := range script.Section(`game\ai0`).Values() {
				names = append(names, name)
			}

			Expect(names).To(Equal([]string{"shortname"}))
		})

		It("returns nothing on a nil section", func() {
			var section *tdf.Section

			Expect(maps.Collect(section.Values())).To(BeEmpty())
		})
	})

	Context("Sections", func() {
		It("returns the sections sorted by name, without values", func() {
			var names []string
			for name := range script.Sections() {
				names = append(names, name)
			}

			Expect(names).To(Equal([]string{`a\b`, "game"}))
		})

		It("returns nothing on a nil section", func() {
			var section *tdf.Section

			Expect(maps.Collect(section.Sections())).To(BeEmpty())
		})
	})

	Context("Indexed", func() {
		It("returns the numbered sections sorted by number", func() {
			var sides []tdf.Value
			var indices []int
			for index, team := range script.Indexed(`game\team`) {
				indices = append(indices, index)
				sides = append(sides, valueAt(team, "side"))
			}

			Expect(indices).To(Equal([]int{0, 2}))
			Expect(sides).To(Equal([]tdf.Value{"Armada", "Cortex"}))
		})

		It("ignores case", func() {
			Expect(maps.Collect(script.Indexed(`GAME\Player`))).To(HaveLen(1))
		})

		It("stops when the loop stops", func() {
			count := 0
			for range script.Indexed(`game\team`) {
				count++
				break
			}

			Expect(count).To(Equal(1))
		})

		It("returns nothing for a missing section", func() {
			Expect(maps.Collect(script.Indexed(`missing\team`))).To(BeEmpty())
		})

		It("returns nothing on a nil section", func() {
			var section *tdf.Section

			Expect(maps.Collect(section.Indexed("team"))).To(BeEmpty())
		})
	})

	Context("MarshalJSON", func() {
		It("writes values as strings and sections as objects", func() {
			encoded, err := json.Marshal(script.Section(`game\ai0`))

			Expect(err).NotTo(HaveOccurred())
			Expect(encoded).To(MatchJSON(`{"shortname":"BARb","options":{"profile":"Hard"}}`))
		})

		It("writes a section held by value", func() {
			holder := struct {
				Script tdf.Section `json:"script"`
			}{Script: *script.Section(`game\ai0`)}

			encoded, err := json.Marshal(holder)

			Expect(err).NotTo(HaveOccurred())
			Expect(encoded).To(MatchJSON(`{"script":{"shortname":"BARb","options":{"profile":"Hard"}}}`))
		})

		It("writes an empty section as an empty object", func() {
			Expect(json.Marshal(tdf.Section{})).To(MatchJSON(`{}`))
		})

		It("writes a nil section as null", func() {
			var section *tdf.Section

			Expect(json.Marshal(section)).To(MatchJSON(`null`))
		})
	})

	Context("UnmarshalJSON", func() {
		It("reads what MarshalJSON writes", func() {
			encoded, err := json.Marshal(script)
			Expect(err).NotTo(HaveOccurred())

			var result tdf.Section
			Expect(json.Unmarshal(encoded, &result)).To(Succeed())

			Expect(json.Marshal(&result)).To(MatchJSON(encoded))
		})

		It("lowercases names", func() {
			var result tdf.Section
			Expect(json.Unmarshal([]byte(`{"A":"x","B":{"C":"y"}}`), &result)).To(Succeed())

			Expect(json.Marshal(&result)).To(MatchJSON(`{"a":"x","b":{"c":"y"}}`))
		})

		It("keeps the last of names which are the same after lowercasing", func() {
			var result tdf.Section
			Expect(json.Unmarshal([]byte(`{"a":"x","A":{"b":"y"},"c":{},"C":"z"}`), &result)).To(Succeed())

			Expect(json.Marshal(&result)).To(MatchJSON(`{"a":{"b":"y"},"c":"z"}`))
		})

		It("replaces the content of the section", func() {
			result := script
			Expect(json.Unmarshal([]byte(`{"a":"x"}`), result)).To(Succeed())

			Expect(json.Marshal(result)).To(MatchJSON(`{"a":"x"}`))
		})

		It("leaves a section held by value unchanged for null", func() {
			holder := struct {
				Script tdf.Section `json:"script"`
			}{Script: *script.Section(`game\ai0`)}

			Expect(json.Unmarshal([]byte(`{"script":null}`), &holder)).To(Succeed())

			Expect(valueAt(&holder.Script, "shortname")).To(Equal(tdf.Value("BARb")))
		})

		It("rejects data after the object", func() {
			var result tdf.Section

			err := result.UnmarshalJSON([]byte(`{"a":"x"} {}`))

			Expect(err).To(MatchError("data after the JSON object"))
		})

		DescribeTable("rejects",
			func(data string, expectedError string) {
				var result tdf.Section

				err := json.Unmarshal([]byte(data), &result)

				Expect(err).To(MatchError(ContainSubstring(expectedError)))
			},
			Entry("an array", `[]`, "want a JSON object"),
			Entry("an empty name", `{"":"x"}`, "empty name"),
			Entry("a number", `{"a":1}`, `"a": want a string or an object`),
			Entry("null", `{"a":null}`, `"a": want a string or an object`),
			Entry("a boolean in a nested section", `{"a":{"b":true}}`, `"a": "b": want a string or an object`),
		)

		Context("MaxDepth", func() {
			var maxDepthBackup int

			BeforeEach(func() {
				maxDepthBackup = tdf.MaxDepth
				tdf.MaxDepth = 2
			})

			AfterEach(func() {
				tdf.MaxDepth = maxDepthBackup
			})

			It("accepts objects nested MaxDepth deep", func() {
				var result tdf.Section

				Expect(json.Unmarshal([]byte(`{"a":{"a":{}}}`), &result)).To(Succeed())
			})

			It("rejects objects nested deeper than MaxDepth", func() {
				var result tdf.Section

				err := json.Unmarshal([]byte(`{"a":{"a":{"a":{}}}}`), &result)

				Expect(err).To(MatchError(ContainSubstring("nested deeper than 2 sections")))
			})
		})
	})
})

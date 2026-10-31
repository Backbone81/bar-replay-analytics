package tdf_test

import (
	"encoding/json"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/tdf"
)

var _ = Describe("Parse", func() {
	DescribeTable("parses",
		func(script string, expectedJSON string) {
			result, err := tdf.Parse(script)
			Expect(err).NotTo(HaveOccurred())

			encoded, err := json.Marshal(result)
			Expect(err).NotTo(HaveOccurred())
			Expect(encoded).To(MatchJSON(expectedJSON))
		},
		Entry("an empty script", "", `{}`),
		Entry("only whitespace and comments", " \t\v\f\n// comment\n/* block */", `{}`),
		Entry("a value", "a=b;", `{"a":"b"}`),
		Entry("blanks around the equals sign", "a \t= \tb;", `{"a":"b"}`),
		Entry("trailing blanks of a value", "a=b \t;", `{"a":"b \t"}`),
		Entry("an empty value", "a=;", `{"a":""}`),
		Entry("several semicolons", "a=b;;;c=d;", `{"a":"b","c":"d"}`),
		Entry("a quoted value", `a="x;y";`, `{"a":"x;y"}`),
		Entry("a quoted value with blanks before the semicolon", `a=" x " ;`, `{"a":" x "}`),
		Entry("quotes followed by more text as an unquoted value", `a="x" y;`, `{"a":"\"x\" y"}`),
		Entry("quotes over two lines as an unquoted value", "a=\"x;\nb=c;", `{"a":"\"x","b":"c"}`),
		Entry("a quote inside an unquoted value", `a=x"y;`, `{"a":"x\"y"}`),
		Entry("a key with special characters", `{a/b.c;"=1;`, `{"{a/b.c;\"":"1"}`),
		Entry("a key starting with a slash", "/a=1;", `{"/a":"1"}`),
		Entry("an unclosed block comment as a key", "/*a=1;", `{"/*a":"1"}`),
		Entry("lowercased keys, but not values", "KeY=VaLuE;", `{"key":"VaLuE"}`),
		Entry("only ASCII lowercased", "ÄB=Ä;[ÖC]{}", `{"Äb":"Ä","Öc":{}}`),
		Entry("a section", "[game]{a=b;}", `{"game":{"a":"b"}}`),
		Entry("nested sections", "[game]\n{\n\t[ai0]\n\t{\n\t\t[options]\n\t\t{\n\t\t\tprofile=Hard;\n\t\t}\n\t}\n}",
			`{"game":{"ai0":{"options":{"profile":"Hard"}}}}`),
		Entry("a lowercased section name with spaces", "[My Section]{}", `{"my section":{}}`),
		Entry("a section name with special characters", "[a{b\nc[d]{}", `{"a{b\nc[d":{}}`),
		Entry("comments between the name and the brace", "[a] // x\n /* y */ {}", `{"a":{}}`),
		Entry("comments between elements", "a=1; // x\n/* y\nz */b=2;", `{"a":"1","b":"2"}`),
		Entry("a comment marker inside a value", "a=b//c;", `{"a":"b//c"}`),
		Entry("the last of two values", "a=1;A=2;", `{"a":"2"}`),
		Entry("a section replacing a value", "a=1;[a]{b=2;}", `{"a":{"b":"2"}}`),
		Entry("a value replacing a section", "[a]{b=2;}a=1;", `{"a":"1"}`),
		Entry("the last of two sections, not merged", "[a]{b=1;}[A]{c=2;}", `{"a":{"c":"2"}}`),
		Entry("CRLF line endings", "[a]\r\n{\r\nb=1;\r\n}\r\n", `{"a":{"b":"1"}}`),
		Entry("a lone CR as a newline in a comment", "// x\ra=1;", `{"a":"1"}`),
		Entry("]] and [[ as written", "a=]][[;", `{"a":"]][["}`),
	)

	DescribeTable("rejects",
		func(script string, expectedLine int, expectedMessage string) {
			result, err := tdf.Parse(script)

			Expect(result).To(BeNil())
			Expect(err).To(MatchError(&tdf.ParseError{Line: expectedLine, Message: expectedMessage}))
		},
		Entry("a key without equals sign", "a", 1, `key "a": missing "="`),
		Entry("a newline before the equals sign", "a\n=b;", 1, `key "a": missing "="`),
		Entry("an equals sign without key", "=b;", 1, `missing key before "="`),
		Entry("a value without semicolon", "a=b", 1, `key "a": missing ";" after the value`),
		Entry("a newline before the semicolon", "a=b\n;", 1, `key "a": missing ";" after the value`),
		Entry("a lone CR before the semicolon", "a=b\r;", 1, `key "a": missing ";" after the value`),
		Entry("a section without closing bracket", "[a{}", 1, `section: missing "]"`),
		Entry("a section with an empty name", "[]{}", 1, "section: empty name"),
		Entry("a section without opening brace", "[a]", 1, `section "a": missing "{"`),
		Entry("a section with text before the opening brace", "[a] b{}", 1, `section "a": missing "{"`),
		Entry("a section without closing brace", "\n[a]{\nb=1;", 2, `section "a": missing "}"`),
		Entry("a closing brace outside of a section", "a=1;\n}", 2, `"}" outside of a section`),
		Entry("an error in a nested section", "[a]{\n[b]{\nc\n}}", 3, `key "c": missing "="`),
	)

	It("rejects a NUL byte", func() {
		result, err := tdf.Parse("a=\x00;")

		Expect(result).To(BeNil())
		Expect(err).To(MatchError("NUL byte at offset 2"))
	})

	It("rejects invalid UTF-8", func() {
		result, err := tdf.Parse("a=\xff;")

		Expect(result).To(BeNil())
		Expect(err).To(MatchError("not valid UTF-8"))
	})

	Context("MaxDepth", func() {
		var maxDepthBackup int

		BeforeEach(func() {
			maxDepthBackup = tdf.MaxDepth
			tdf.MaxDepth = 3
		})

		AfterEach(func() {
			tdf.MaxDepth = maxDepthBackup
		})

		It("accepts sections nested MaxDepth deep", func() {
			Expect(tdf.Parse(nested(3))).Error().NotTo(HaveOccurred())
		})

		It("rejects sections nested deeper than MaxDepth", func() {
			Expect(tdf.Parse(nested(4))).Error().To(MatchError(`line 1: section "a": nested deeper than 3 sections`))
		})
	})

	It("parses a setup script of a demo", func() {
		result, err := tdf.Parse(testScript)
		Expect(err).NotTo(HaveOccurred())

		Expect(valueAt(result, `game\mapname`)).To(Equal(tdf.Value("Argent Strata 1.1")))
		Expect(valueAt(result, `game\ai0\options\profile`)).To(Equal(tdf.Value("Hard")))
		Expect(valueAt(result, `game\team0\rgbcolor`)).To(Equal(tdf.Value("0.99609375 0.546875 0")))
		Expect(valueAt(result, `game\modoptions\date_year`)).To(Equal(tdf.Value("2026")))
	})
})

// valueAt returns the value at location in section and expects it to be there.
func valueAt(section *tdf.Section, location string) tdf.Value {
	value, ok := section.Value(location)
	ExpectWithOffset(1, ok).To(BeTrue(), "value at %s", location)
	return value
}

// nested returns depth sections named "a", each inside the one before.
func nested(depth int) string {
	return strings.Repeat("[a]{", depth) + strings.Repeat("}", depth)
}

// testScript is the setup script of a demo of a 1v1 against BARb, as the engine writes it.
const testScript = `[game]
{
[allyteam1]
{
startrectleft=0.74000001;
startrectright=0.88;
startrectbottom=0.28;
numallies=0;
startrecttop=0.115;
}
[team0]
{
allyteam=0;
teamleader=0;
rgbcolor=0.99609375 0.546875 0;
side=Cortex;
handicap=0;
}
[allyteam0]
{
startrectleft=0.12;
startrectright=0.25999999;
startrectbottom=0.88499999;
numallies=0;
startrecttop=0.72000003;
}
[modoptions]
{
date_year=2026;
date_day=24;
date_month=10;
date_hour=19;
}
[player0]
{
rank=0;
team=0;
name=Player;
isfromdemo=0;
}
[ai0]
{
[options]
{
profile=Hard;
}
shortname=BARb;
team=1;
host=0;
name=BARbarianAI(1);
isfromdemo=0;
}
[team1]
{
allyteam=1;
teamleader=0;
rgbcolor=0.99609375 0.546875 0;
side=Random;
handicap=0;
}
ishost=1;
hostip=127.0.0.1;
numusers=2;
startpostype=2;
gametype=Beyond All Reason test-31723-95ab750;
mapname=Argent Strata 1.1;
myplayername=Player;
numplayers=1;
gamestartdelay=5;
nohelperais=0;
hostport=0;
}
`

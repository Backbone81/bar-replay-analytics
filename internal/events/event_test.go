package events_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/events"
)

var _ = Describe("Event", func() {
	Context("Validate", func() {
		It("accepts an event with one type", func() {
			event := events.Event{Frame: 30, UnitFinished: &events.UnitFinished{UnitID: 1}}

			Expect(event.Validate()).To(Succeed())
		})

		It("accepts an event without a type, of a type it does not know", func() {
			event := events.Event{Frame: 30}

			Expect(event.Validate()).To(Succeed())
		})

		It("rejects an event with two types", func() {
			event := events.Event{
				Frame:        30,
				UnitCreated:  &events.UnitCreated{UnitID: 1},
				UnitFinished: &events.UnitFinished{UnitID: 1},
			}

			Expect(event.Validate()).NotTo(Succeed())
		})
	})

	Context("UnitDefIDs", func() {
		It("returns the unit def of a unit event", func() {
			event := events.Event{UnitGiven: &events.UnitGiven{UnitDefID: 101}}

			Expect(event.UnitDefIDs()).To(Equal([]int{101}))
		})

		It("returns the unit defs of the unit and its attacker", func() {
			event := events.Event{UnitDestroyed: &events.UnitDestroyed{UnitDefID: 101, AttackerDefID: new(102)}}

			Expect(event.UnitDefIDs()).To(Equal([]int{101, 102}))
		})

		It("returns the unit def a builder builds", func() {
			event := events.Event{BuilderState: &events.BuilderState{CmdID: new(-102)}}

			Expect(event.UnitDefIDs()).To(Equal([]int{102}))
		})

		It("returns none for a builder's other commands", func() {
			event := events.Event{BuilderState: &events.BuilderState{CmdID: new(90)}}

			Expect(event.UnitDefIDs()).To(BeEmpty())
		})

		It("returns none for team resources", func() {
			event := events.Event{TeamResources: &events.TeamResources{}}

			Expect(event.UnitDefIDs()).To(BeEmpty())
		})
	})
})

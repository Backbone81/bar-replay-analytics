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
})

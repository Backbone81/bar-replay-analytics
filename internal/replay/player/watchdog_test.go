package player_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	replayplayer "github.com/backbone81/bar-replay-analytics/internal/replay/player"
	"github.com/backbone81/bar-replay-analytics/internal/replay/player/widget"
)

var _ = Describe("Watchdog", func() {
	Context("Write", func() {
		It("takes all of the output", func() {
			watchdog := replayplayer.NewWatchdog(time.Minute, time.Minute)
			output := "[f=0000000] engine output\nwithout a line end"

			Expect(watchdog.Write([]byte(output))).To(Equal(len(output)))
		})
	})

	Context("Watch", func() {
		It("cancels when the analytics widget does not become active in time", func(ctx SpecContext) {
			watchdog := replayplayer.NewWatchdog(100*time.Millisecond, time.Minute)

			watchedCtx := watchInBackground(ctx, watchdog)
			writeLine(watchdog, "[f=-000001] Loading widget from user:  Replay Analytics")

			Eventually(context.Cause).WithArguments(watchedCtx).
				Should(MatchError(replayplayer.ErrAnalyticsWidgetNotActive))
		})

		It("cancels when the heartbeat stops", func(ctx SpecContext) {
			watchdog := replayplayer.NewWatchdog(time.Minute, 100*time.Millisecond)

			watchedCtx := watchInBackground(ctx, watchdog)
			writeLine(watchdog, "[f=-000001] "+widget.ActiveLogLine)

			Eventually(context.Cause).WithArguments(watchedCtx).
				Should(MatchError(replayplayer.ErrAnalyticsWidgetHeartbeatMissing))
		})

		It("does not cancel while the heartbeat comes", func(ctx SpecContext) {
			watchdog := replayplayer.NewWatchdog(200*time.Millisecond, 200*time.Millisecond)

			watchedCtx := watchInBackground(ctx, watchdog)
			writeLine(watchdog, "[f=-000001] "+widget.ActiveLogLine)
			for frame := range 20 {
				time.Sleep(25 * time.Millisecond)
				writeLine(watchdog, "[f=0000000] "+widget.HeartbeatLogLine+" (frame 0)")
				Expect(watchedCtx.Err()).ToNot(HaveOccurred(), "after heartbeat %d", frame)
			}
		})

		It("recognizes a log line written in parts", func(ctx SpecContext) {
			watchdog := replayplayer.NewWatchdog(200*time.Millisecond, time.Minute)

			watchedCtx := watchInBackground(ctx, watchdog)
			Expect(watchdog.Write([]byte("[f=-000001] Replay Anal"))).To(Equal(23))
			Expect(watchdog.Write([]byte("ytics: active\n"))).To(Equal(14))

			Consistently(watchedCtx.Err).WithTimeout(400 * time.Millisecond).ShouldNot(HaveOccurred())
		})

		It("recognizes a log line among several lines written at once", func(ctx SpecContext) {
			watchdog := replayplayer.NewWatchdog(200*time.Millisecond, time.Minute)
			output := "[f=-000001] engine output\n[f=-000001] " + widget.ActiveLogLine + "\n[f=-000001] more output\n"

			watchedCtx := watchInBackground(ctx, watchdog)
			Expect(watchdog.Write([]byte(output))).To(Equal(len(output)))

			Consistently(watchedCtx.Err).WithTimeout(400 * time.Millisecond).ShouldNot(HaveOccurred())
		})

		It("does not count a log line before its end", func(ctx SpecContext) {
			watchdog := replayplayer.NewWatchdog(100*time.Millisecond, time.Minute)

			watchedCtx := watchInBackground(ctx, watchdog)
			Expect(watchdog.Write([]byte("[f=-000001] " + widget.ActiveLogLine))).To(Equal(36))

			Eventually(context.Cause).WithArguments(watchedCtx).
				Should(MatchError(replayplayer.ErrAnalyticsWidgetNotActive))
		})

		It("returns without canceling when its context is done", func(ctx SpecContext) {
			watchdog := replayplayer.NewWatchdog(time.Minute, time.Minute)
			cancelableCtx, cancel := context.WithCancel(ctx)
			causes := newCauseRecorder()
			cancel()

			watchdog.Watch(cancelableCtx, causes.Cancel)

			Expect(causes).ToNot(Receive())
		})
	})
})

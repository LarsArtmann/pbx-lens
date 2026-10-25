package web

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

func TestDashboardQualityRollupUsesItsOwnDenominators(t *testing.T) {
	t.Parallel()

	started := testCallStarted()

	// A: transcript with a user turn, both parties, ends 5 s after the turn.
	withUser := webObservation("conv-with-user", started)
	withUser.Transcript = []calls.TranscriptTurn{
		{Role: calls.TurnRoleUser, Text: "hello", At: started},
		{Role: calls.TurnRoleAssistant, Text: "hi", At: started.Add(time.Second)},
	}
	withUser.DurationSecs = 5
	withUser.HangupCause = "NORMAL_CLEARING"

	// B: transcript with no user turn and a long unlogged tail.
	noUser := webObservation("conv-no-user", started.Add(time.Hour))
	noUser.Transcript = []calls.TranscriptTurn{
		{Role: calls.TurnRoleAssistant, Text: "hello?", At: started.Add(time.Hour)},
	}
	noUser.DurationSecs = 44
	noUser.HangupCause = "NORMAL_CLEARING"

	// C: no transcript at all, and both parties identical (self-dial).
	selfDial := webObservation("conv-self", started.Add(2*time.Hour))
	selfDial.Transcript = nil
	selfDial.ToNumber = selfDial.FromNumber
	selfDial.DurationSecs = 3
	selfDial.HangupCause = "CALL_REJECTED"

	stats, err := DashboardStatsQuery(
		t.Context(),
		newTestSystem(t, withUser, noUser, selfDial),
		started.Add(3*time.Hour),
		ListCallsQuery{},
		dashboardWindowDays,
	)
	if err != nil {
		t.Fatalf("DashboardStatsQuery: %v", err)
	}

	rollup := stats.Quality
	if rollup.Calls != 3 {
		t.Errorf("calls = %d, want 3", rollup.Calls)
	}

	// Only the two calls that had a transcript can fail "no user turn".
	if rollup.Transcripts != 2 || rollup.NoUserTurn != 1 {
		t.Errorf("no-user-turn = %d/%d, want 1/2 transcripts", rollup.NoUserTurn, rollup.Transcripts)
	}

	if rollup.PartiesKnown != 3 || rollup.SelfDial != 1 {
		t.Errorf("self-dial = %d/%d, want 1/3", rollup.SelfDial, rollup.PartiesKnown)
	}

	// Both transcripts carry a start-of-call turn; only B runs far past it.
	if rollup.TailEligible != 2 || rollup.RecordingTail != 1 {
		t.Errorf("recording tail = %d/%d, want 1/2 timed transcripts", rollup.RecordingTail, rollup.TailEligible)
	}

	if len(rollup.HangupCauses) != 2 || rollup.HangupCauses[0].Cause != "NORMAL_CLEARING" ||
		rollup.HangupCauses[0].Calls != 2 {
		t.Errorf("hangup causes = %+v, want NORMAL_CLEARING x2 first", rollup.HangupCauses)
	}
}

func TestDashboardQualityRollupFoldsTheLongTail(t *testing.T) {
	t.Parallel()

	started := testCallStarted()
	seeded := make([]calls.Observation, 0, hangupCauseLimit+3)

	for i := range hangupCauseLimit + 3 {
		obs := webObservation("conv-cause-"+string(rune('a'+i)), started.Add(time.Duration(i)*time.Minute))
		obs.HangupCause = "CAUSE_" + string(rune('A'+i))
		seeded = append(seeded, obs)
	}

	stats, err := DashboardStatsQuery(
		t.Context(),
		newTestSystem(t, seeded...),
		started.Add(time.Hour),
		ListCallsQuery{},
		dashboardWindowDays,
	)
	if err != nil {
		t.Fatalf("DashboardStatsQuery: %v", err)
	}

	if len(stats.Quality.HangupCauses) != hangupCauseLimit+1 {
		t.Fatalf("causes = %d rows, want %d + one folded entry", len(stats.Quality.HangupCauses), hangupCauseLimit)
	}

	last := stats.Quality.HangupCauses[len(stats.Quality.HangupCauses)-1]
	if last.Cause != "other" || last.Calls != 3 {
		t.Errorf("folded entry = %+v, want other x3", last)
	}
}

func TestDashboardRendersQualitySignals(t *testing.T) {
	t.Parallel()

	started := testCallStarted()
	noUser := webObservation("conv-quiet", started)
	noUser.Transcript = []calls.TranscriptTurn{{Role: calls.TurnRoleAssistant, Text: "hello?"}}

	_, body := getBody(t, newTestServer(t, noUser), "/dashboard")

	for _, want := range []string{"How calls end", "Hangup causes", "Source truths", "No user turn", "1 transcript", "100.0%"} {
		if !strings.Contains(body, want) {
			t.Errorf("quality card missing %q", want)
		}
	}
}

func TestQualityEmptyWindowIsHonest(t *testing.T) {
	t.Parallel()

	_, body := getBody(t, newTestServer(t), "/dashboard")

	if !strings.Contains(body, "What calls end with nothing indexed") &&
		!strings.Contains(body, "No calls in the window yet.") {
		t.Error("an empty window must explain itself, not render 0% rates")
	}

	if strings.Contains(body, `self-dial`) {
		t.Error("empty window must not render signal rows")
	}
}

func TestQualityTrendMovesInPoints(t *testing.T) {
	t.Parallel()

	// 14 days: prior week 10 transcripts, 5 without a user turn (50%);
	// this week 10 transcripts, 2 without (20%) → −30 points.
	days := make([]DayVolume, 0, 2*weekDays)
	for i := range 2 * weekDays {
		day := DayVolume{Day: "2026-10-" + strconv.Itoa(i+1), Transcripts: 10}
		if i < weekDays {
			day.NoUserTurn = 5
		} else {
			day.NoUserTurn = 2
		}

		days = append(days, day)
	}

	trend := qualityTrend(days)
	if !trend.HasBaseline || trend.Points != -30 {
		t.Errorf("qualityTrend = %+v, want HasBaseline with -30 points", trend)
	}

	// No transcripts in the prior week → no baseline, never a fabricated move.
	for i := range days[:weekDays] {
		days[i].Transcripts = 0
		days[i].NoUserTurn = 0
	}

	if trend := qualityTrend(days); trend.HasBaseline {
		t.Errorf("qualityTrend = %+v, want no baseline without prior transcripts", trend)
	}

	// A prior week with a zero share has no meaningful move either (a
	// percentage from zero is drama, not signal).
	for i := range days[:weekDays] {
		days[i].Transcripts = 10
		days[i].NoUserTurn = 0
	}

	if trend := qualityTrend(days); trend.HasBaseline {
		t.Errorf("qualityTrend = %+v, want no baseline from a zero prior share", trend)
	}

	// Short windows cannot compare weeks at all.
	if trend := qualityTrend(days[:weekDays]); trend.HasBaseline {
		t.Errorf("qualityTrend = %+v, want no baseline from a short window", trend)
	}
}

package analysis

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// fakeMarkdown is the canned response for the dev fake mode. Non-trivial size,
// real markdown features (headings, table, lists, bold, code) so the FE
// renderer gets exercised. Used by streamFakeAnalysis and streamFakeFollowUp.
const fakeMarkdown = `## Portfolio Snapshot

Your portfolio currently spans **5 asset classes** across 12 instruments. Here's a structured analysis based on the latest snapshot.

### Allocation overview

| Class       | Allocation | Notional   | Trend   |
|-------------|-----------:|-----------:|---------|
| Equities    |     58.4 % | $145,200   | Up      |
| Crypto      |     19.1 % |  $47,500   | Flat    |
| Fixed Income|     12.7 % |  $31,600   | Down    |
| Cash        |      6.3 % |  $15,700   | Stable  |
| Commodities |      3.5 % |   $8,700   | Up      |

### Key observations

1. **Concentration risk** — your top 3 equity positions account for 41% of total NAV. A 10% drawdown in any one of them moves the portfolio by ~4%.
2. **Crypto exposure is elevated** for a balanced strategy. The 90-day realized vol on this sleeve is ~62%, contributing roughly 35% of the portfolio's overall variance despite being only 19% of NAV.
3. **Cash drag** — at 6.3%, idle cash is reasonable, but consider laddered T-bills for the portion you don't need liquid.
4. **Fixed income drift** — duration has crept up over the last quarter; check if that aligns with your rate outlook.

### Risk metrics (trailing 90d)

- **Sharpe ratio:** 1.18 (annualized, vs. 0.94 for the 60/40 benchmark).
- **Max drawdown:** -8.7% on March 14 (recovered in 11 sessions).
- **Beta vs. SPX:** 0.87 — slightly less volatile than the broad market, mostly thanks to fixed income.
- **Correlation to BTC:** 0.41 (rising; a year ago it was 0.18).

### Suggested moves

> These are *observations*, not recommendations. Always pair with your own thesis.

- Trim the top equity position by 25% to bring it under the 15% single-name threshold.
- Rebalance crypto to target 12-15% rather than the current 19% — the upside has been captured; the asymmetry now favors trimming.
- Move 40% of idle cash to a 4-week T-bill ladder; expected pickup ~ 90 bps net.
- Audit fixed-income duration: if you're targeting < 4 years, the current 5.2 is offside.

### Tax-efficiency notes

` + "```" + `
Realized YTD gain:  $4,300
Unrealized gain:    $18,700  (long-term: $14,200, short-term: $4,500)
Harvestable losses: $1,200   (concentrated in 2 names)
` + "```" + `

You have some room to harvest losses against the short-term gains. The two losers in your tracker have been held for 70+ days — close to the 30-day wash-sale window if you want to repurchase.

### What to monitor next

- The Fed meeting on the 5th — your fixed-income sleeve will react sharply to a hawkish surprise.
- Earnings from the top equity position (week of the 12th) — this is your single largest concentration.
- Stablecoin yields — if they slip below T-bill yields again, the rationale for the crypto cash leg weakens.

Let me know if you want a deeper dive on any of these threads, or if you'd like me to run scenarios against a hypothetical reallocation.
`

// fakeFollowUpMarkdown is shorter and simulates a follow-up reply.
const fakeFollowUpMarkdown = `Good question. A few angles to consider:

1. **Position sizing.** If a name has rallied past your target weight, the trim isn't a vote of no confidence — it's mechanical risk management. The thesis can stay intact.
2. **Tax friction.** A 25% trim of a long-term holding triggers LTCG; pair it with a harvestable lossmaker if you have one.
3. **Re-entry plan.** Decide *now* what would make you re-add: a price level, an earnings beat, a multiple compression. Otherwise the trimmed capital tends to sit in cash indefinitely.

If you want, I can sketch out a specific trim sequence with limit prices.
`

// streamFakeAnalysis emits a synthetic SSE response that mimics the backend's
// initial-analysis stream: one session event, many delta events chunked from
// fakeMarkdown, then a done event. Used when ANALYSIS_FAKE=true.
func streamFakeAnalysis(c *gin.Context) {
	streamFake(c, fakeMarkdown, true)
}

// streamFakeFollowUp emits a synthetic SSE response for a follow-up message.
// No session event (the FE already has session_id).
func streamFakeFollowUp(c *gin.Context) {
	streamFake(c, fakeFollowUpMarkdown, false)
}

// streamFake is the shared emitter. emitSession controls whether the first
// event is a `session` event (true for the initial stream, false for
// follow-ups).
func streamFake(c *gin.Context, markdown string, emitSession bool) {
	setSSEHeaders(c)
	c.Status(http.StatusOK)
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return
	}

	if emitSession {
		sessionID := "fake-" + randomHex(8)
		payload, _ := json.Marshal(map[string]string{"session_id": sessionID})
		fmt.Fprintf(c.Writer, "event: session\ndata: %s\n\n", payload)
		flusher.Flush()
	}

	const chunkSize = 18
	const chunkDelay = 30 * time.Millisecond

	chunks := chunkifyMarkdown(markdown, chunkSize)
	for _, chunk := range chunks {
		select {
		case <-c.Request.Context().Done():
			return
		default:
		}
		payload, _ := json.Marshal(map[string]string{"text": chunk})
		if _, err := fmt.Fprintf(c.Writer, "event: delta\ndata: %s\n\n", payload); err != nil {
			return
		}
		flusher.Flush()
		time.Sleep(chunkDelay)
	}

	fmt.Fprintf(c.Writer, "event: done\ndata: {}\n\n")
	flusher.Flush()
}

// chunkifyMarkdown splits the input into chunks of approximately approxSize
// runes, trying to break on a space when one is near so we don't slice in the
// middle of a token. Returns a non-empty slice for non-empty input.
func chunkifyMarkdown(s string, approxSize int) []string {
	if s == "" {
		return nil
	}
	runes := []rune(s)
	out := make([]string, 0, len(runes)/approxSize+1)
	for i := 0; i < len(runes); {
		end := i + approxSize
		if end >= len(runes) {
			out = append(out, string(runes[i:]))
			break
		}
		// Look ahead a few runes for a whitespace to break on; keeps tokens
		// (like inline code or **bold**) intact in most cases.
		broken := false
		for j := end; j < len(runes) && j < end+8; j++ {
			if runes[j] == ' ' || runes[j] == '\n' {
				out = append(out, string(runes[i:j+1]))
				i = j + 1
				broken = true
				break
			}
		}
		if !broken {
			out = append(out, string(runes[i:end]))
			i = end
		}
	}
	return out
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return strings.Repeat("0", n*2)
	}
	return hex.EncodeToString(buf)
}

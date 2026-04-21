package context

// sliceAroundAnchor keeps up to maxMsgs messages from msgs (already oldest-first)
// centered on anchorTS when set. If anchorTS is empty, returns msgs unchanged.
func sliceAroundAnchor(msgs []slackMessage, anchorTS string, maxMsgs int) []slackMessage {
	if anchorTS == "" || len(msgs) == 0 {
		return msgs
	}
	as, an, okA := parseSlackTSNs(anchorTS)
	if !okA {
		return msgs
	}
	bestIdx := -1
	for i := range msgs {
		s, n, ok := parseSlackTSNs(msgs[i].TS)
		if !ok {
			continue
		}
		if s < as || (s == as && n <= an) {
			bestIdx = i
		}
	}
	if bestIdx < 0 {
		return msgs
	}
	start := bestIdx - (maxMsgs-1)/2
	if start < 0 {
		start = 0
	}
	end := start + maxMsgs
	if end > len(msgs) {
		end = len(msgs)
		start = end - maxMsgs
		if start < 0 {
			start = 0
		}
	}
	return msgs[start:end]
}

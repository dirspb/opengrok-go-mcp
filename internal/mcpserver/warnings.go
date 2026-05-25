// SPDX-License-Identifier: Apache-2.0

package mcpserver

// appendWarning joins msg onto an existing warning with a single space and
// returns the combined pointer. An empty msg returns existing unchanged; a
// nil/empty existing yields just msg. This replaces the ad-hoc warning
// concatenation that was duplicated across search paths.
func appendWarning(existing *string, msg string) *string {
	if msg == "" {
		return existing
	}
	if existing == nil || *existing == "" {
		return &msg
	}
	combined := *existing + " " + msg
	return &combined
}

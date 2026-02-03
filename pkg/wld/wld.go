package wld

// Match reports whether string s matches pattern p.
// '*' matches any sequence of runes (including empty).
// '?' matches any single rune.
// This function treats strings as rune sequences so it works with Unicode.
func Match(p, s string) bool {
	pr := []rune(p)
	sr := []rune(s)
	pi, si := 0, 0
	starIdx := -1 // index of last '*' in pattern
	match := 0    // index in sr corresponding to position after last star consumed

	for si < len(sr) {
		// Use a switch over boolean conditions to avoid an if-else chain (gocritic: ifElseChain).
		switch {
		case pi < len(pr) && (pr[pi] == '?' || pr[pi] == sr[si]):
			pi++
			si++
		case pi < len(pr) && pr[pi] == '*':
			// record star position and the position in sr where star started matching
			starIdx = pi
			match = si
			pi++
		case starIdx != -1:
			// backtrack: let last star match one more rune
			pi = starIdx + 1
			match++
			si = match
		default:
			return false
		}
	}

	// skip trailing '*' in pattern
	for pi < len(pr) && pr[pi] == '*' {
		pi++
	}

	return pi == len(pr)
}

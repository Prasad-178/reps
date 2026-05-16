package agents

// Categories the Planner may choose from.
var Categories = []string{
	"system-design",
	"domain-crypto",
	"domain-ml",
	"domain-solana",
	"jd-specific",
	"general",
}

func IsValidCategory(c string) bool {
	for _, k := range Categories {
		if k == c {
			return true
		}
	}
	return false
}

package elo

import "math"

// RatingToScore maps a 1-5 judge rating to an Elo "score" in [0,1].
//   1 → 0.0   2 → 0.25   3 → 0.5   4 → 0.75   5 → 1.0
func RatingToScore(rating int) float64 {
	switch rating {
	case 1:
		return 0.0
	case 2:
		return 0.25
	case 3:
		return 0.5
	case 4:
		return 0.75
	case 5:
		return 1.0
	}
	if rating < 1 {
		return 0.0
	}
	return 1.0
}

// Expected computes the expected score (probability of "winning") given the
// candidate's rating and the question's difficulty rating.
func Expected(userRating, questionRating int) float64 {
	exponent := float64(questionRating-userRating) / 400.0
	return 1.0 / (1.0 + math.Pow(10, exponent))
}

// Update returns the new user rating after the candidate scored `score` on a
// question of difficulty `questionRating`. K=24 by default (slower than chess).
func Update(userRating, questionRating int, score float64, k int) (newRating int, delta int) {
	if k <= 0 {
		k = 24
	}
	expected := Expected(userRating, questionRating)
	deltaF := float64(k) * (score - expected)
	d := int(math.Round(deltaF))
	return userRating + d, d
}

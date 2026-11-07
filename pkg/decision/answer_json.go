package decision

import "encoding/json"

// MarshalJSON writes exactly the fields of the answer's type, zero values
// included: a noul answer of probability 0 must not lose its probability to
// omitempty.
func (a Answer) MarshalJSON() ([]byte, error) {
	switch a.Type {
	case Noul:
		return json.Marshal(struct {
			Type        Type    `json:"type"`
			Probability float64 `json:"probability"`
		}{a.Type, a.Probability})
	case Choice:
		return json.Marshal(struct {
			Type          Type               `json:"type"`
			Choice        string             `json:"choice"`
			Confidence    float64            `json:"confidence"`
			Probabilities map[string]float64 `json:"probabilities,omitempty"`
		}{a.Type, a.Choice, a.Confidence, a.Probabilities})
	case Score:
		return json.Marshal(struct {
			Type               Type      `json:"type"`
			Score              float64   `json:"score"`
			Confidence         float64   `json:"confidence"`
			LevelProbabilities []float64 `json:"level_probabilities,omitempty"`
		}{a.Type, a.Score, a.Confidence, a.LevelProbabilities})
	}
	type plain Answer
	return json.Marshal(plain(a))
}

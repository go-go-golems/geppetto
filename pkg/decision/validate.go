package decision

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

// tolerance absorbs floating-point rounding in provider probabilities.
const tolerance = 1e-6

// ValidateRequest checks a request against the question model and the
// provider's configured model. Errors wrap ErrInvalidRequest and name only
// question ids, never state or question text.
func ValidateRequest(in Request, configured Model) error {
	if in.Model != "" && configured.Name != "" && in.Model != configured.Name {
		return fmt.Errorf("decision request model %q does not match the provider model %q: %w", in.Model, configured.Name, ErrInvalidRequest)
	}
	if err := checkContent(in.State); err != nil {
		return fmt.Errorf("decision state %s: %w", err, ErrInvalidRequest)
	}
	for i, img := range in.Images {
		if !isImageDataURL(img) {
			return fmt.Errorf("decision image %d must be a data:image/{jpeg,png,webp};base64 URL: %w", i, ErrInvalidRequest)
		}
	}
	if len(in.Questions) == 0 {
		return fmt.Errorf("decision request needs at least one question: %w", ErrInvalidRequest)
	}
	for _, id := range sortedIDs(in.Questions) {
		if err := validateQuestionID(id); err != nil {
			return err
		}
		if err := validateQuestion(id, in.Questions[id]); err != nil {
			return err
		}
	}
	return nil
}

// ValidateQuestion checks one question (exported for callers that build
// questions incrementally).
func ValidateQuestion(id string, q Question) error {
	if err := validateQuestionID(id); err != nil {
		return err
	}
	return validateQuestion(id, q)
}

func validateQuestionID(id string) error {
	if id == "" || len(id) > MaxQuestionID {
		return fmt.Errorf("decision question id must be 1-%d bytes: %w", MaxQuestionID, ErrInvalidRequest)
	}
	for _, r := range id {
		if !idRune(r) {
			return fmt.Errorf("decision question id may contain only [A-Za-z0-9_.-]: %w", ErrInvalidRequest)
		}
	}
	return nil
}

func idRune(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '.' || r == '-'
}

func validateQuestion(id string, q Question) error {
	bad := func(format string, args ...any) error {
		return fmt.Errorf("decision question %q: %s: %w", id, fmt.Sprintf(format, args...), ErrInvalidRequest)
	}
	if err := checkContent(q.Instructions); err != nil {
		return bad("instructions %s", err)
	}
	hasNoul := q.True != "" || q.False != ""
	switch q.Type {
	case Noul:
		if len(q.Options) > 0 || len(q.Levels) > 0 {
			return bad("a noul question takes only true/false descriptions")
		}
	case Choice:
		if hasNoul || len(q.Levels) > 0 {
			return bad("a choice question takes only options")
		}
		if len(q.Options) == 0 || len(q.Options) > MaxOptions {
			return bad("a choice question needs 1-%d options, got %d", MaxOptions, len(q.Options))
		}
		seen := make(map[string]struct{}, len(q.Options))
		for i, o := range q.Options {
			if strings.TrimSpace(o.Name) == "" {
				return bad("option %d has no name", i)
			}
			if _, dup := seen[o.Name]; dup {
				return bad("option %d duplicates an earlier option name", i)
			}
			seen[o.Name] = struct{}{}
		}
	case Score:
		if hasNoul || len(q.Options) > 0 {
			return bad("a score question takes only levels")
		}
		if len(q.Levels) == 0 || len(q.Levels) > MaxLevels {
			return bad("a score question needs 1-%d levels, got %d", MaxLevels, len(q.Levels))
		}
		for i, l := range q.Levels {
			if strings.TrimSpace(l) == "" {
				return bad("level %d is empty", i)
			}
		}
	default:
		return bad("unknown type %q (noul, choice, score)", q.Type)
	}
	return nil
}

// checkContent accepts a non-empty string or JSON-encodable structured data
// (an object or array). It returns a short reason, never the content.
func checkContent(v any) error {
	if v == nil {
		return fmt.Errorf("is required")
	}
	if s, ok := v.(string); ok {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("is empty")
		}
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("is not JSON-encodable")
	}
	switch {
	case len(b) == 0:
		return fmt.Errorf("is empty")
	case b[0] == '{', b[0] == '[':
		if string(b) == "{}" || string(b) == "[]" {
			return fmt.Errorf("is empty")
		}
		return nil
	case b[0] == '"':
		var s string
		if json.Unmarshal(b, &s) == nil && strings.TrimSpace(s) != "" {
			return nil
		}
		return fmt.Errorf("is empty")
	default:
		return fmt.Errorf("must be a string, object or array")
	}
}

func isImageDataURL(s string) bool {
	for _, p := range []string{"data:image/jpeg;base64,", "data:image/png;base64,", "data:image/webp;base64,"} {
		if strings.HasPrefix(s, p) && len(s) > len(p) {
			return true
		}
	}
	return false
}

// ValidateResponse checks answers against the request: exactly one answer
// per question, of the question's type, with finite probabilities over
// exactly the declared answer space. Errors wrap ErrInvalidResponse.
func ValidateResponse(in Request, answers map[string]Answer) error {
	for _, id := range sortedIDs(in.Questions) {
		if _, ok := answers[id]; !ok {
			return fmt.Errorf("decision response has no answer for question %q: %w", id, ErrInvalidResponse)
		}
	}
	for _, id := range sortedAnswerIDs(answers) {
		q, ok := in.Questions[id]
		if !ok {
			return fmt.Errorf("decision response answers a question that was not asked: %w", ErrInvalidResponse)
		}
		if err := validateAnswer(q, answers[id]); err != nil {
			return fmt.Errorf("decision answer %q: %s: %w", id, err, ErrInvalidResponse)
		}
	}
	return nil
}

func validateAnswer(q Question, a Answer) error {
	if a.Type != q.Type {
		return fmt.Errorf("type %q, asked %q", a.Type, q.Type)
	}
	switch q.Type {
	case Noul:
		return checkProbability("probability", a.Probability)
	case Choice:
		if err := checkProbability("confidence", a.Confidence); err != nil {
			return err
		}
		names := make(map[string]struct{}, len(q.Options))
		for _, o := range q.Options {
			names[o.Name] = struct{}{}
		}
		if _, ok := names[a.Choice]; !ok {
			return fmt.Errorf("choice is not one of the options")
		}
		for name, p := range a.Probabilities {
			if _, ok := names[name]; !ok {
				return fmt.Errorf("probability for an undeclared option")
			}
			if err := checkProbability("option probability", p); err != nil {
				return err
			}
		}
	case Score:
		if err := checkProbability("confidence", a.Confidence); err != nil {
			return err
		}
		top := float64(len(q.Levels) - 1)
		if math.IsNaN(a.Score) || math.IsInf(a.Score, 0) || a.Score < -tolerance || a.Score > top+tolerance {
			return fmt.Errorf("score %v outside [0, %v]", a.Score, top)
		}
		if len(a.LevelProbabilities) != len(q.Levels) {
			return fmt.Errorf("%d level probabilities for %d levels", len(a.LevelProbabilities), len(q.Levels))
		}
		for _, p := range a.LevelProbabilities {
			if err := checkProbability("level probability", p); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkProbability(what string, p float64) error {
	if math.IsNaN(p) || math.IsInf(p, 0) || p < -tolerance || p > 1+tolerance {
		return fmt.Errorf("%s %v outside [0, 1]", what, p)
	}
	return nil
}

func sortedIDs(m map[string]Question) []string {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func sortedAnswerIDs(m map[string]Answer) []string {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

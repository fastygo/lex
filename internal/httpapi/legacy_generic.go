package httpapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fastygo/lex/internal/canonical"
	"github.com/fastygo/lex/internal/evidence"
	"github.com/fastygo/lex/internal/profile"
	"github.com/fastygo/lex/internal/verify"
	"github.com/fastygo/lex/internal/wire"
)

// verifyLegacyDecisionStructure runs the deprecated claim route through the
// generic State + QuestionSet structural verifier before its compatibility
// policy adds a Verdict. Its temporary bundle is never returned or retained.
func verifyLegacyDecisionStructure(
	ctx context.Context,
	prof profile.Profile,
	body evaluationRequest,
	frozen evidence.Frozen,
	items []evidence.Item,
	decider Decider,
	decision Decision,
	verifier wire.DecisionVerifier,
) error {
	state, err := json.Marshal(prof.State(body.Query, items))
	if err != nil {
		return fmt.Errorf("encode compatibility state: %w", err)
	}
	contextRecord, err := legacyContextRecord(frozen)
	if err != nil {
		return err
	}
	bundle, err := wire.BuildDecisionBundle(wire.DecisionBundleInput{
		ProjectID: body.ProjectID,
		Decision:  wire.DecisionIdentity{ID: "compatibility", Version: "1"},
		State:     state, QuestionSet: legacyQuestionSet(prof),
		Context:   contextRecord,
		AdapterID: decider.AdapterID(), AdapterVersion: decider.AdapterVersion(),
		ResolvedModel: decision.ResolvedModel, Answers: decision.Answers,
	})
	if err != nil {
		return fmt.Errorf("seal generic compatibility decision: %w", err)
	}
	_, err = verifier.ReplayContext(ctx, bundle)
	return err
}

func legacyQuestionSet(prof profile.Profile) wire.GenericQuestionSet {
	legacy := prof.QuestionSet()
	questions := make(map[string]wire.GenericQuestion, len(legacy.Questions))
	for id, question := range legacy.Questions {
		converted := wire.GenericQuestion{Type: verify.QuestionType(question.Type), Instructions: question.Instructions}
		if converted.Type == verify.QuestionChoice {
			converted.Options = question.Criteria
		}
		questions[id] = converted
	}
	return wire.GenericQuestionSet{ID: legacy.ID, Version: legacy.Version, Questions: questions}
}

func legacyContextRecord(frozen evidence.Frozen) (*wire.ContextRecord, error) {
	snapshot, err := json.Marshal(frozen.Snapshot)
	if err != nil {
		return nil, fmt.Errorf("encode frozen snapshot: %w", err)
	}
	request, err := json.Marshal(frozen.PackRequest)
	if err != nil {
		return nil, fmt.Errorf("encode frozen pack request: %w", err)
	}
	hash, err := canonical.HashJSON(frozen.Pack)
	if err != nil {
		return nil, fmt.Errorf("hash frozen pack: %w", err)
	}
	return &wire.ContextRecord{Pack: frozen.Pack, Snapshot: snapshot, PackRequest: request, PackHash: hash}, nil
}

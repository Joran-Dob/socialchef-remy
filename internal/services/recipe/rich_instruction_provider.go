package recipe

import "context"

type RichInstructionProvider interface {
	GenerateRichInstructions(ctx context.Context, recipe *Recipe) (*RichInstructionResponse, error)
}

type RichInstructionResponse struct {
	Instructions  []RichInstruction `json:"instructions"`
	PromptVersion int               `json:"-"`
}

type RichInstruction struct {
	StepNumber      int    `json:"step_number"`
	InstructionRich string `json:"instruction_rich"`
}

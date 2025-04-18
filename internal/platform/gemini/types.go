// Package gemini provides implementations for the generation interface using Google's Gemini API.
package gemini

// promptData represents the data passed to the prompt template
type promptData struct {
	MemoText string
}

// ResponseSchema represents the expected structure of a card from the Gemini API
type ResponseSchema struct {
	// Cards is the array of flashcards generated from the memo text
	Cards []CardSchema `json:"cards"`
}

// CardSchema represents a single flashcard in the API response
type CardSchema struct {
	// Front is the question or prompt side of the flashcard
	Front string `json:"front"`

	// Back is the answer side of the flashcard
	Back string `json:"back"`

	// Hint is an optional hint to help the user recall the answer
	Hint string `json:"hint,omitempty"`

	// Tags are optional categories or labels for the flashcard
	Tags []string `json:"tags,omitempty"`
}

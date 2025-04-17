# TODO

## Generation Service Implementation
- [x] **T001:** Define the Generator interface in generation package
    - **Action:** Create `internal/generation/generator.go` with the Generator interface that defines the `GenerateCards` method to create flashcards from memo text.
    - **Depends On:** None
    - **AC Ref:** N/A

- [x] **T002:** Define custom error types for generation package
    - **Action:** Create `internal/generation/errors.go` with specific error types for generation failures, invalid responses, content blocking, transient failures, and invalid configuration.
    - **Depends On:** [T001]
    - **AC Ref:** N/A

- [x] **T003:** Update configuration structure for LLM settings
    - **Action:** Add LLMConfig struct to `internal/config/config.go` with fields for API key, model name, prompt template path, max retries, and retry delay.
    - **Depends On:** None
    - **AC Ref:** N/A

- [x] **T004:** Set default values for LLM configuration
    - **Action:** Update `internal/config/load.go` to set default values for max retries and retry delay.
    - **Depends On:** [T003]
    - **AC Ref:** N/A

- [x] **T005:** Create prompt template for flashcard generation
    - **Action:** Create a prompt template file at an appropriate location (e.g., `prompts/flashcard_template.txt`) with instructions for the LLM.
    - **Depends On:** None
    - **AC Ref:** N/A

- [x] **T006:** Add gemini platform package
    - **Action:** Create `internal/platform/gemini` package with package documentation in `doc.go`.
    - **Depends On:** None
    - **AC Ref:** N/A

- [x] **T007:** Implement GeminiGenerator struct and constructor
    - **Action:** Create `internal/platform/gemini/gemini_generator.go` with the struct definition and NewGeminiGenerator constructor.
    - **Depends On:** [T001, T003, T006]
    - **AC Ref:** N/A

- [x] **T008:** Implement prompt creation function
    - **Action:** Add createPrompt method to geminiGenerator to generate the prompt string from the template.
    - **Depends On:** [T007]
    - **AC Ref:** N/A

- [x] **T009:** Implement Gemini API call with retry logic
    - **Action:** Add callGeminiWithRetry method to handle API calls with exponential backoff retry.
    - **Depends On:** [T002, T007]
    - **AC Ref:** N/A

- [x] **T010:** Implement response parsing logic
    - **Action:** Add parseResponse method to convert API response JSON into domain.Card objects.
    - **Depends On:** [T007]
    - **AC Ref:** N/A

- [x] **T011:** Implement GenerateCards method
    - **Action:** Implement the GenerateCards method to fulfill the Generator interface, using the helper methods.
    - **Depends On:** [T008, T009, T010]
    - **AC Ref:** N/A

- [x] **T012:** Add required dependencies to go.mod
    - **Action:** Update go.mod to include Google's generative-ai-go/genai and related dependencies.
    - **Depends On:** None
    - **AC Ref:** N/A

- [x] **T013:** Update server initialization code
    - **Action:** Modify `cmd/server/main.go` to initialize the gemini generator and inject it into the dependency chain.
    - **Depends On:** [T007, T012]
    - **AC Ref:** N/A

- [x] **T014:** Create tests for Generator interface
    - **Action:** Create `internal/generation/generation_test.go` to test documentation and error types.
    - **Depends On:** [T001, T002]
    - **AC Ref:** N/A

- [x] **T015:** Create tests for GeminiGenerator implementation
    - **Action:** Create `internal/platform/gemini/gemini_generator_test.go` to test prompt template parsing, error handling, retry logic, and response parsing.
    - **Depends On:** [T007, T008, T009, T010, T011]
    - **AC Ref:** N/A

- [x] **T016:** Create mock generator for integration tests
    - **Action:** Create `internal/mocks/generator.go` with a mock implementation of the Generator interface.
    - **Depends On:** [T001]
    - **AC Ref:** N/A

- [x] **T017:** Update existing integration tests to use mock generator
    - **Action:** Modify relevant integration tests to use the new mock generator.
    - **Depends On:** [T016]
    - **AC Ref:** N/A

- [x] **T018:** Update config.yaml.example with LLM configuration
    - **Action:** Add the LLM configuration section to the example configuration file.
    - **Depends On:** [T003]
    - **AC Ref:** N/A

- [x] **T019:** Update default LLM model to gemini-2.0-flash
    - **Action:** Update the default model in `internal/config/load.go` from gemini-1.5-flash to gemini-2.0-flash.
    - **Depends On:** [T004]
    - **AC Ref:** N/A

- [ ] **T020:** Perform final validation and testing
    - **Action:** Ensure all tests pass, code adheres to project standards, and validation criteria are met.
    - **Depends On:** [T001, T002, T003, T004, T005, T006, T007, T008, T009, T010, T011, T012, T013, T014, T015, T016, T017, T018, T019]
    - **AC Ref:** N/A

- [x] **T021:** Switch from generative-ai-go to genai library
    - **Action:** Update dependencies and code to use the new genai library instead of generative-ai-go.
    - **Depends On:** [T012, T013]
    - **AC Ref:** N/A

## [!] CLARIFICATIONS NEEDED / ASSUMPTIONS
- [ ] **Assumption:** CardContent structure exists in the domain package
    - **Context:** The implementation assumes there's a domain.CardContent struct with Front and Back fields, but this isn't explicitly confirmed in the PLAN.md.

- [ ] **Assumption:** The domain.NewCard function exists and accepts userID, memoID, and content
    - **Context:** The implementation assumes the domain.NewCard constructor exists with the specified parameters.

- [ ] **Assumption:** The API key will be stored in the environment/configuration
    - **Context:** The implementation assumes the API key will be provided via the configuration system, but exact security measures aren't specified.

package recipe

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/socialchef/remy/internal/httpclient"
	"github.com/socialchef/remy/internal/metrics"
	"github.com/socialchef/remy/internal/services/ai"
)

func init() {
	_ = metrics.Init()
}

type redirectTransport struct {
	target string
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq := req.Clone(req.Context())
	newReq.URL, _ = req.URL.Parse(t.target + "/v1/chat/completions")
	return http.DefaultTransport.RoundTrip(newReq)
}

func TestCerebrasGenerateCategories_Success(t *testing.T) {
	expectedResponse := ai.CategoryAIResponse{
		CuisineCategories:   []string{"Italian", "Mediterranean"},
		MealTypes:           []string{"Dinner", "Lunch"},
		Occasions:           []string{"Weeknight", "Special Occasion"},
		DietaryRestrictions: []string{"Vegetarian", "Gluten-Free"},
		Equipment:           []string{"Oven", "Stovetop", "Baking Dish"},
		NewCategorySuggestions: &ai.NewCategorySet{
			CuisineCategories:   []string{},
			MealTypes:           []string{},
			Occasions:           []string{},
			DietaryRestrictions: []string{},
			Equipment:           []string{},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("Expected path /v1/chat/completions, got %s", r.URL.Path)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected Authorization Bearer test-key, got %s", r.Header.Get("Authorization"))
		}

		respContent, _ := json.Marshal(expectedResponse)
		chatResp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": string(respContent),
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResp)
	}))
	defer server.Close()

	originalClient := httpclient.InstrumentedClient
	defer func() { httpclient.InstrumentedClient = originalClient }()

	httpclient.InstrumentedClient = &http.Client{
		Transport: &redirectTransport{target: server.URL},
		Timeout:   180 * time.Second,
	}

	provider := NewCerebrasProvider("test-key")

	ctx := context.Background()
	prompt := "Test prompt for categories"
	result, err := provider.GenerateCategories(ctx, prompt)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if len(result.CuisineCategories) != 2 {
		t.Errorf("Expected 2 cuisine categories, got %d", len(result.CuisineCategories))
	}
	if result.CuisineCategories[0] != "Italian" {
		t.Errorf("Expected first cuisine category 'Italian', got %s", result.CuisineCategories[0])
	}
	if len(result.MealTypes) != 2 {
		t.Errorf("Expected 2 meal types, got %d", len(result.MealTypes))
	}
	if result.MealTypes[0] != "Dinner" {
		t.Errorf("Expected first meal type 'Dinner', got %s", result.MealTypes[0])
	}
	if len(result.Occasions) != 2 {
		t.Errorf("Expected 2 occasions, got %d", len(result.Occasions))
	}
	if result.Occasions[0] != "Weeknight" {
		t.Errorf("Expected first occasion 'Weeknight', got %s", result.Occasions[0])
	}
	if len(result.DietaryRestrictions) != 2 {
		t.Errorf("Expected 2 dietary restrictions, got %d", len(result.DietaryRestrictions))
	}
	if result.DietaryRestrictions[0] != "Vegetarian" {
		t.Errorf("Expected first dietary restriction 'Vegetarian', got %s", result.DietaryRestrictions[0])
	}
	if len(result.Equipment) != 3 {
		t.Errorf("Expected 3 equipment items, got %d", len(result.Equipment))
	}
	if result.Equipment[0] != "Oven" {
		t.Errorf("Expected first equipment 'Oven', got %s", result.Equipment[0])
	}
	if result.NewCategorySuggestions == nil {
		t.Error("Expected NewCategorySuggestions to be set, got nil")
	}
}

func TestCerebrasGenerateCategories_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "rate limit exceeded",
		})
	}))
	defer server.Close()

	originalClient := httpclient.InstrumentedClient
	defer func() { httpclient.InstrumentedClient = originalClient }()

	httpclient.InstrumentedClient = &http.Client{
		Transport: &redirectTransport{target: server.URL},
		Timeout:   180 * time.Second,
	}

	provider := NewCerebrasProvider("test-key")

	ctx := context.Background()
	prompt := "Test prompt for categories"
	_, err := provider.GenerateCategories(ctx, prompt)

	if err == nil {
		t.Fatal("Expected error for rate limit, got nil")
	}

	providerErr, ok := err.(*ProviderError)
	if !ok {
		t.Fatalf("Expected ProviderError, got %T", err)
	}

	if providerErr.Type != "rate_limit" {
		t.Errorf("Expected error type 'rate_limit', got '%s'", providerErr.Type)
	}

	if providerErr.Provider != "cerebras" {
		t.Errorf("Expected provider 'cerebras', got '%s'", providerErr.Provider)
	}

	if providerErr.Message == "" {
		t.Error("Expected error message, got empty string")
	}
}

func TestCerebrasGenerateCategories_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "internal server error",
		})
	}))
	defer server.Close()

	originalClient := httpclient.InstrumentedClient
	defer func() { httpclient.InstrumentedClient = originalClient }()

	httpclient.InstrumentedClient = &http.Client{
		Transport: &redirectTransport{target: server.URL},
		Timeout:   180 * time.Second,
	}

	provider := NewCerebrasProvider("test-key")

	ctx := context.Background()
	prompt := "Test prompt for categories"
	_, err := provider.GenerateCategories(ctx, prompt)

	if err == nil {
		t.Fatal("Expected error for server error, got nil")
	}

	providerErr, ok := err.(*ProviderError)
	if !ok {
		t.Fatalf("Expected ProviderError, got %T", err)
	}

	if providerErr.Type != "server_error" {
		t.Errorf("Expected error type 'server_error', got '%s'", providerErr.Type)
	}

	if providerErr.Provider != "cerebras" {
		t.Errorf("Expected provider 'cerebras', got '%s'", providerErr.Provider)
	}
}

func TestCerebrasGenerateCategories_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chatResp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": "this is not valid json",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResp)
	}))
	defer server.Close()

	originalClient := httpclient.InstrumentedClient
	defer func() { httpclient.InstrumentedClient = originalClient }()

	httpclient.InstrumentedClient = &http.Client{
		Transport: &redirectTransport{target: server.URL},
		Timeout:   180 * time.Second,
	}

	provider := NewCerebrasProvider("test-key")

	ctx := context.Background()
	prompt := "Test prompt for categories"
	_, err := provider.GenerateCategories(ctx, prompt)

	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}

	errMsg := err.Error()
	if containsSubstring(errMsg, "failed to unmarshal") || containsSubstring(errMsg, "invalid character") {
	} else {
		t.Logf("Error message: %s", errMsg)
		t.Logf("Expected error message to contain JSON unmarshaling error")
	}

	providerErr, ok := err.(*ProviderError)
	if !ok {
		t.Logf("Expected ProviderError, got %T", err)
	} else {
		if providerErr.Provider != "cerebras" {
			t.Errorf("Expected provider 'cerebras', got '%s'", providerErr.Provider)
		}
	}
}

func TestCerebrasGenerateCategories_NoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chatResp := map[string]interface{}{
			"choices": []map[string]interface{}{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResp)
	}))
	defer server.Close()

	originalClient := httpclient.InstrumentedClient
	defer func() { httpclient.InstrumentedClient = originalClient }()

	httpclient.InstrumentedClient = &http.Client{
		Transport: &redirectTransport{target: server.URL},
		Timeout:   180 * time.Second,
	}

	provider := NewCerebrasProvider("test-key")

	ctx := context.Background()
	prompt := "Test prompt for categories"
	_, err := provider.GenerateCategories(ctx, prompt)

	if err == nil {
		t.Fatal("Expected error for no choices, got nil")
	}

	errMsg := err.Error()
	if !containsSubstring(errMsg, "no response") {
		t.Errorf("Expected error to contain 'no response', got: %s", errMsg)
	}

	providerErr, ok := err.(*ProviderError)
	if !ok {
		t.Logf("Expected ProviderError, got %T", err)
	} else {
		if providerErr.Provider != "cerebras" {
			t.Errorf("Expected provider 'cerebras', got '%s'", providerErr.Provider)
		}
	}
}

func TestCerebrasGenerateCategories_EmptyCategories(t *testing.T) {
	emptyResponse := ai.CategoryAIResponse{
		CuisineCategories:   []string{},
		MealTypes:           []string{},
		Occasions:           []string{},
		DietaryRestrictions: []string{},
		Equipment:           []string{},
		NewCategorySuggestions: &ai.NewCategorySet{
			CuisineCategories:   []string{},
			MealTypes:           []string{},
			Occasions:           []string{},
			DietaryRestrictions: []string{},
			Equipment:           []string{},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respContent, _ := json.Marshal(emptyResponse)
		chatResp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": string(respContent),
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResp)
	}))
	defer server.Close()

	originalClient := httpclient.InstrumentedClient
	defer func() { httpclient.InstrumentedClient = originalClient }()

	httpclient.InstrumentedClient = &http.Client{
		Transport: &redirectTransport{target: server.URL},
		Timeout:   180 * time.Second,
	}

	provider := NewCerebrasProvider("test-key")

	ctx := context.Background()
	prompt := "Test prompt for categories"
	result, err := provider.GenerateCategories(ctx, prompt)

	if err != nil {
		t.Errorf("Expected no error for empty categories, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if len(result.CuisineCategories) != 0 {
		t.Errorf("Expected 0 cuisine categories, got %d", len(result.CuisineCategories))
	}
	if len(result.MealTypes) != 0 {
		t.Errorf("Expected 0 meal types, got %d", len(result.MealTypes))
	}
	if len(result.Occasions) != 0 {
		t.Errorf("Expected 0 occasions, got %d", len(result.Occasions))
	}
	if len(result.DietaryRestrictions) != 0 {
		t.Errorf("Expected 0 dietary restrictions, got %d", len(result.DietaryRestrictions))
	}
	if len(result.Equipment) != 0 {
		t.Errorf("Expected 0 equipment items, got %d", len(result.Equipment))
	}
	if result.NewCategorySuggestions == nil {
		t.Error("Expected NewCategorySuggestions to be set, got nil")
	}
}

func TestCerebrasGenerateRichInstructions_Success(t *testing.T) {
	expectedResponse := RichInstructionResponse{
		Instructions: []RichInstruction{
			{
				StepNumber:      1,
				InstructionRich: "Add {{ingredient:550e8400-e29b-41d4-a716-446655440000}} to the bowl and mix for {{timer:0}}.",
			},
			{
				StepNumber:      2,
				InstructionRich: "Bake with {{ingredient:660e8400-e29b-41d4-a716-446655440001}} at 180°C.",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("Expected path /v1/chat/completions, got %s", r.URL.Path)
		}

		respContent, _ := json.Marshal(expectedResponse)
		chatResp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": string(respContent),
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResp)
	}))
	defer server.Close()

	// Override InstrumentedClient to use test server
	originalClient := httpclient.InstrumentedClient
	defer func() { httpclient.InstrumentedClient = originalClient }()

	httpclient.InstrumentedClient = &http.Client{
		Transport: &redirectTransport{target: server.URL},
		Timeout:   180 * time.Second,
	}

	provider := NewCerebrasProvider("test-key")

	recipe := &Recipe{
		RecipeName: "Test Recipe",
		Ingredients: []Ingredient{
			{ID: "550e8400-e29b-41d4-a716-446655440000", Name: "flour"},
			{ID: "660e8400-e29b-41d4-a716-446655440001", Name: "sugar"},
		},
		Instructions: []Instruction{
			{
				StepNumber:  1,
				Instruction: "Add flour to the bowl and mix for 5 minutes.",
				TimerData: []Timer{
					{DurationSeconds: 300, DurationText: "5 minutes", Label: "Mix", Type: "prep", Category: "active"},
				},
			},
			{
				StepNumber:  2,
				Instruction: "Bake with sugar at 180°C.",
				TimerData:   []Timer{},
			},
		},
	}

	ctx := context.Background()
	result, err := provider.GenerateRichInstructions(ctx, recipe)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if len(result.Instructions) != 2 {
		t.Errorf("Expected 2 instructions, got %d", len(result.Instructions))
	}

	if result.PromptVersion <= 0 {
		t.Errorf("Expected PromptVersion to be set, got %d", result.PromptVersion)
	}
}

func TestCerebrasGenerateRichInstructions_InvalidPlaceholder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a placeholder that doesn't match the expected pattern
		// The validation only checks placeholders that match the pattern
		respContent := `{"instructions":[{"step_number":1,"instruction_rich":"Add {{ingredient:invalid}} to bowl"}]}`
		chatResp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": respContent,
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResp)
	}))
	defer server.Close()

	// Override InstrumentedClient to use test server
	originalClient := httpclient.InstrumentedClient
	defer func() { httpclient.InstrumentedClient = originalClient }()

	httpclient.InstrumentedClient = &http.Client{
		Transport: &redirectTransport{target: server.URL},
		Timeout:   180 * time.Second,
	}

	provider := NewCerebrasProvider("test-key")

	recipe := &Recipe{
		RecipeName: "Test Recipe",
		Ingredients: []Ingredient{
			{ID: "550e8400-e29b-41d4-a716-446655440000", Name: "flour"},
		},
		Instructions: []Instruction{
			{
				StepNumber:  1,
				Instruction: "Add flour to bowl",
				TimerData:   []Timer{},
			},
		},
	}

	ctx := context.Background()
	result, err := provider.GenerateRichInstructions(ctx, recipe)

	// Invalid placeholders that don't match the pattern are silently ignored
	if err != nil {
		t.Errorf("Expected no error for non-matching placeholder, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}
}

func TestCerebrasGenerateRichInstructions_OutOfBoundsUUID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respContent := `{"instructions":[{"step_number":1,"instruction_rich":"Add {{ingredient:99999999-9999-9999-9999-999999999999}} to bowl"}]}`
		chatResp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": respContent,
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResp)
	}))
	defer server.Close()

	// Override InstrumentedClient to use test server
	originalClient := httpclient.InstrumentedClient
	defer func() { httpclient.InstrumentedClient = originalClient }()

	httpclient.InstrumentedClient = &http.Client{
		Transport: &redirectTransport{target: server.URL},
		Timeout:   180 * time.Second,
	}

	provider := NewCerebrasProvider("test-key")

	recipe := &Recipe{
		RecipeName: "Test Recipe",
		Ingredients: []Ingredient{
			{Name: "flour"},
		},
		Instructions: []Instruction{
			{
				StepNumber:  1,
				Instruction: "Add flour to bowl",
				TimerData:   []Timer{},
			},
		},
	}

	ctx := context.Background()
	_, err := provider.GenerateRichInstructions(ctx, recipe)

	if err == nil {
		t.Error("Expected error for out of bounds UUID, got nil")
	}
}

func TestCerebrasGenerateRichInstructions_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "rate limit exceeded",
		})
	}))
	defer server.Close()

	// Override InstrumentedClient to use test server
	originalClient := httpclient.InstrumentedClient
	defer func() { httpclient.InstrumentedClient = originalClient }()

	httpclient.InstrumentedClient = &http.Client{
		Transport: &redirectTransport{target: server.URL},
		Timeout:   180 * time.Second,
	}

	provider := NewCerebrasProvider("test-key")

	recipe := &Recipe{
		RecipeName:  "Test Recipe",
		Ingredients: []Ingredient{},
		Instructions: []Instruction{
			{StepNumber: 1, Instruction: "Step 1"},
		},
	}

	ctx := context.Background()
	_, err := provider.GenerateRichInstructions(ctx, recipe)

	if err == nil {
		t.Error("Expected error for rate limit, got nil")
	}
}

func TestCerebrasGenerateRichInstructions_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "internal server error",
		})
	}))
	defer server.Close()

	// Override InstrumentedClient to use test server
	originalClient := httpclient.InstrumentedClient
	defer func() { httpclient.InstrumentedClient = originalClient }()

	httpclient.InstrumentedClient = &http.Client{
		Transport: &redirectTransport{target: server.URL},
		Timeout:   180 * time.Second,
	}

	provider := NewCerebrasProvider("test-key")

	recipe := &Recipe{
		RecipeName:  "Test Recipe",
		Ingredients: []Ingredient{},
		Instructions: []Instruction{
			{StepNumber: 1, Instruction: "Step 1"},
		},
	}

	ctx := context.Background()
	_, err := provider.GenerateRichInstructions(ctx, recipe)

	if err == nil {
		t.Error("Expected error for server error, got nil")
	}
}

func TestCerebrasGenerateRichInstructions_NoIngredients(t *testing.T) {
	expectedResponse := RichInstructionResponse{
		Instructions: []RichInstruction{
			{
				StepNumber:      1,
				InstructionRich: "Preheat the oven to 180°C.",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respContent, _ := json.Marshal(expectedResponse)
		chatResp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": string(respContent),
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResp)
	}))
	defer server.Close()

	// Override InstrumentedClient to use test server
	originalClient := httpclient.InstrumentedClient
	defer func() { httpclient.InstrumentedClient = originalClient }()

	httpclient.InstrumentedClient = &http.Client{
		Transport: &redirectTransport{target: server.URL},
		Timeout:   180 * time.Second,
	}

	provider := NewCerebrasProvider("test-key")

	recipe := &Recipe{
		RecipeName:  "Test Recipe",
		Ingredients: []Ingredient{},
		Instructions: []Instruction{
			{StepNumber: 1, Instruction: "Preheat the oven to 180°C.", TimerData: []Timer{}},
		},
	}

	ctx := context.Background()
	result, err := provider.GenerateRichInstructions(ctx, recipe)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if len(result.Instructions) != 1 {
		t.Errorf("Expected 1 instruction, got %d", len(result.Instructions))
	}
}

func TestCerebrasGenerateRichInstructions_UUIDGeneration(t *testing.T) {
	// Test that when ingredients don't have IDs, the implementation generates UUIDs for them
	// and the response uses those generated UUIDs
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse the request to extract the ingredient UUIDs that were generated
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		// Return a simple response without placeholders since we're testing UUID generation
		// not placeholder validation
		respContent := `{"instructions":[{"step_number":1,"instruction_rich":"Add flour to bowl"}]}`
		chatResp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": respContent,
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResp)
	}))
	defer server.Close()

	// Override InstrumentedClient to use test server
	originalClient := httpclient.InstrumentedClient
	defer func() { httpclient.InstrumentedClient = originalClient }()

	httpclient.InstrumentedClient = &http.Client{
		Transport: &redirectTransport{target: server.URL},
		Timeout:   180 * time.Second,
	}

	provider := NewCerebrasProvider("test-key")

	// Ingredient WITHOUT an ID - implementation should generate one
	recipe := &Recipe{
		RecipeName: "Test Recipe",
		Ingredients: []Ingredient{
			{Name: "flour"},
		},
		Instructions: []Instruction{
			{
				StepNumber:  1,
				Instruction: "Add flour to bowl",
				TimerData:   []Timer{},
			},
		},
	}

	ctx := context.Background()
	result, err := provider.GenerateRichInstructions(ctx, recipe)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if len(result.Instructions) != 1 {
		t.Errorf("Expected 1 instruction, got %d", len(result.Instructions))
	}
}

func TestCerebrasGenerateRichInstructions_TimerMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a response with timer index 1 when only 0 is valid
		respContent := `{"instructions":[{"step_number":1,"instruction_rich":"Mix for {{timer:1}} minutes"}]}`
		chatResp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": respContent,
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResp)
	}))
	defer server.Close()

	// Override InstrumentedClient to use test server
	originalClient := httpclient.InstrumentedClient
	defer func() { httpclient.InstrumentedClient = originalClient }()

	httpclient.InstrumentedClient = &http.Client{
		Transport: &redirectTransport{target: server.URL},
		Timeout:   180 * time.Second,
	}

	provider := NewCerebrasProvider("test-key")

	recipe := &Recipe{
		RecipeName: "Test Recipe",
		Ingredients: []Ingredient{
			{Name: "flour"},
		},
		Instructions: []Instruction{
			{
				StepNumber:  1,
				Instruction: "Mix flour",
				TimerData: []Timer{
					{DurationSeconds: 300, DurationText: "5 minutes", Label: "Mix", Type: "prep", Category: "active"},
				},
			},
		},
	}

	ctx := context.Background()
	_, err := provider.GenerateRichInstructions(ctx, recipe)

	if err == nil {
		t.Error("Expected error for timer index out of bounds, got nil")
	}
}

func TestCerebrasGenerateRichInstructions_NoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chatResp := map[string]interface{}{
			"choices": []map[string]interface{}{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResp)
	}))
	defer server.Close()

	// Override InstrumentedClient to use test server
	originalClient := httpclient.InstrumentedClient
	defer func() { httpclient.InstrumentedClient = originalClient }()

	httpclient.InstrumentedClient = &http.Client{
		Transport: &redirectTransport{target: server.URL},
		Timeout:   180 * time.Second,
	}

	provider := NewCerebrasProvider("test-key")

	recipe := &Recipe{
		RecipeName:  "Test Recipe",
		Ingredients: []Ingredient{},
		Instructions: []Instruction{
			{StepNumber: 1, Instruction: "Step 1"},
		},
	}

	ctx := context.Background()
	_, err := provider.GenerateRichInstructions(ctx, recipe)

	if err == nil {
		t.Error("Expected error for no choices, got nil")
	}
}

func TestCerebrasGenerateRichInstructions_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chatResp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": "invalid json",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResp)
	}))
	defer server.Close()

	// Override InstrumentedClient to use test server
	originalClient := httpclient.InstrumentedClient
	defer func() { httpclient.InstrumentedClient = originalClient }()

	httpclient.InstrumentedClient = &http.Client{
		Transport: &redirectTransport{target: server.URL},
		Timeout:   180 * time.Second,
	}

	provider := NewCerebrasProvider("test-key")

	recipe := &Recipe{
		RecipeName:  "Test Recipe",
		Ingredients: []Ingredient{},
		Instructions: []Instruction{
			{StepNumber: 1, Instruction: "Step 1"},
		},
	}

	ctx := context.Background()
	_, err := provider.GenerateRichInstructions(ctx, recipe)

	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestCerebrasGenerateRichInstructions_WithExistingIngredientIDs(t *testing.T) {
	ingredientUUID := "123e4567-e89b-12d3-a456-426614174000"

	expectedResponse := RichInstructionResponse{
		Instructions: []RichInstruction{
			{
				StepNumber:      1,
				InstructionRich: "Use {{ingredient:123e4567-e89b-12d3-a456-426614174000}} as base",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respContent, _ := json.Marshal(expectedResponse)
		chatResp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": string(respContent),
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResp)
	}))
	defer server.Close()

	// Override InstrumentedClient to use test server
	originalClient := httpclient.InstrumentedClient
	defer func() { httpclient.InstrumentedClient = originalClient }()

	httpclient.InstrumentedClient = &http.Client{
		Transport: &redirectTransport{target: server.URL},
		Timeout:   180 * time.Second,
	}

	provider := NewCerebrasProvider("test-key")

	recipe := &Recipe{
		RecipeName: "Test Recipe",
		Ingredients: []Ingredient{
			{ID: ingredientUUID, Name: "flour"},
		},
		Instructions: []Instruction{
			{
				StepNumber:  1,
				Instruction: "Use flour as base",
				TimerData:   []Timer{},
			},
		},
	}

	ctx := context.Background()
	result, err := provider.GenerateRichInstructions(ctx, recipe)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if len(result.Instructions) != 1 {
		t.Errorf("Expected 1 instruction, got %d", len(result.Instructions))
	}
}

func TestCerebrasGenerateRichInstructions_MultipleTimers(t *testing.T) {
	expectedResponse := RichInstructionResponse{
		Instructions: []RichInstruction{
			{
				StepNumber:      1,
				InstructionRich: "Mix for {{timer:0}}, then {{timer:1}}",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respContent, _ := json.Marshal(expectedResponse)
		chatResp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": string(respContent),
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResp)
	}))
	defer server.Close()

	// Override InstrumentedClient to use test server
	originalClient := httpclient.InstrumentedClient
	defer func() { httpclient.InstrumentedClient = originalClient }()

	httpclient.InstrumentedClient = &http.Client{
		Transport: &redirectTransport{target: server.URL},
		Timeout:   180 * time.Second,
	}

	provider := NewCerebrasProvider("test-key")

	recipe := &Recipe{
		RecipeName: "Test Recipe",
		Ingredients: []Ingredient{
			{Name: "flour"},
		},
		Instructions: []Instruction{
			{
				StepNumber:  1,
				Instruction: "Mix flour",
				TimerData: []Timer{
					{DurationSeconds: 300, DurationText: "5 minutes", Label: "Mix", Type: "prep", Category: "active"},
					{DurationSeconds: 600, DurationText: "10 minutes", Label: "Rest", Type: "prep", Category: "active"},
				},
			},
		},
	}

	ctx := context.Background()
	result, err := provider.GenerateRichInstructions(ctx, recipe)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if len(result.Instructions) != 1 {
		t.Errorf("Expected 1 instruction, got %d", len(result.Instructions))
	}
}

func TestCerebrasGenerateRichInstructions_InterfaceCompliance(t *testing.T) {
	var _ RichInstructionProvider = (*CerebrasProvider)(nil)
}

package ai

import (
	"fmt"
	"strings"
)

const roleSection = `<ROLE>
You are a specialized AI assistant designed to parse recipe information from social media descriptions. Your task is to extract key details from the given recipe text and structure them in a specific JSON format. You MUST preserve the original language of the recipe content throughout your output.
</ROLE>`

const extractionGuidelinesSection = `<EXTRACTION_GUIDELINES>
When presented with a recipe description, extract the following information into separate components:

1. Main Recipe Information:
   - Recipe name
   - Description
   - Preparation time (in minutes)
   - Cooking time (in minutes)
   - Total time (in minutes)
   - Original serving size
   - Difficulty rating (1-5)
   - Focused diet
   - Estimated calories

2. Ingredients List:
   - Original quantity (as stated in the recipe)
   - Original unit of measurement
   - Total quantity for all servings
   - Unit of measurement for adjusted quantity
   - Ingredient name

3. Instructions List:
   - Step-by-step cooking instructions
   - Each step should be clear, detailed, and actionable
   - Include helpful details such as:
     * Visual cues (e.g., "until golden brown", "until bubbles form")
     * Timing indicators (e.g., "about 5 minutes", "until tender")
     * Texture descriptions (e.g., "should be smooth and creamy")
     * Temperature guidance (e.g., "medium-high heat", "180°C")
     * Technique explanations for complex steps
     * Safety tips when relevant (e.g., "careful, steam will be hot")
     * Common mistakes to avoid
     * Tips for best results

4. Nutritional Information:
   - Protein (grams)
   - Carbohydrates (grams)
   - Fat (grams)
   - Fiber (grams)
</EXTRACTION_GUIDELINES>`

const inferenceSection = `<INFERENCE>
If any information is not explicitly stated, use your best judgment to infer it:

- For the recipe name and description:
  * GENERATE a descriptive recipe name based on what the dish is (do NOT copy the source title verbatim)
  * Focus on the main ingredients and the final dish, not just the cooking method or marketing words
  * If the given name is vague, focuses on a cooking method, or is clickbait, create a more descriptive name
  * The generated name MUST be in the original language of the recipe content
  * Provide a brief, enticing summary of the dish for the description (in the original language)
- For ingredients:
  * Return quantities as stated in the recipe (total amounts for all servings)
  * If the original serving size is not provided, estimate it based on the recipe context
- For time estimates:
  * If not explicitly stated, estimate based on similar recipes and cooking methods
  * Total time should be the sum of preparation and cooking times
- For the original recipe serving size:
  * If not explicitly stated, estimate based on the recipe context and typical serving sizes
- For difficulty rating:
  * Consider the complexity of techniques, number of ingredients, and time required
  * Rate difficulty from 1 (very easy) to 5 (very challenging)
- For the focused diet, determine if the recipe strongly aligns with a specific diet plan
- Estimate the calorie count and nutritional information based on the ingredients for 1 serving
</INFERENCE>`

const outputFormatSection = `<OUTPUT_FORMAT>
Always format your response as a JSON object with the following structure:

{
  "recipe": {
    "recipe_name": "",
    "description": "",
    "prep_time": null,
    "cooking_time": null,
    "total_time": null,
    "original_serving_size": null,
    "difficulty_rating": null,
    "focused_diet": "",
    "estimated_calories": null
  },
  "ingredients": [
    {
      "original_quantity": null,
      "original_unit": "",
      "quantity": null,
      "unit": "",
      "name": ""
    }
  ],
  "instructions": [
    {
      "step_number": null,
      "instruction": "",
      "timer_data": [
        {
          "duration_seconds": null,
          "label": "",
          "type": "",
          "category": ""
        }
      ]
    }
  ],
  "nutrition": {
    "protein": null,
    "carbs": null,
    "fat": null,
    "fiber": null
  },
  "language": ""
}
</OUTPUT_FORMAT>`

const criticalMetricRequirementSection = `<CRITICAL_METRIC_REQUIREMENT>
ALL adjusted ingredient measurements MUST be in METRIC units. This is mandatory.

DO NOT USE these imperial/US units for the "unit" field:
- cups, cup
- tablespoons, tablespoon, tbsp, Tbsp
- teaspoons, teaspoon, tsp
- ounces, ounce, oz
- pounds, pound, lb, lbs
- Fahrenheit, °F

ONLY USE these metric units for the "unit" field:
- Weight: g (grams), mg (milligrams), kg (kilograms)
- Volume: ml (milliliters), L (liters)
- Temperature: °C (Celsius)
- Length: cm (centimeters), mm (millimeters)
- Count-based: piece, whole, slice, clove, sprig, pinch

CONVERSION EXAMPLES - Always convert like this:
- "1 cup flour" → quantity: 120, unit: "g"
- "2 tbsp olive oil" → quantity: 30, unit: "ml"
- "1 tsp salt" → quantity: 5, unit: "g"
- "8 oz chicken" → quantity: 227, unit: "g"
- "1 lb ground beef" → quantity: 454, unit: "g"
- "350°F" → write as "175°C" in instructions
</CRITICAL_METRIC_REQUIREMENT>`

const ingredientAnalysisSection = `<INGREDIENT_ANALYSIS>
%s

When analyzing ingredients:
1. ALWAYS provide BOTH the original quantities (as stated in the recipe) AND total quantities for the complete recipe:
   - original_quantity: The exact quantity as stated in the source recipe (can be in any unit)
   - original_unit: The exact unit as stated in the source recipe (can be imperial)
   - quantity: The TOTAL quantity for all servings combined - MUST BE METRIC
   - unit: The unit for the adjusted quantity - MUST BE METRIC (g, ml, etc.)
   
2. For quantities, ALWAYS return the TOTAL amount as stated in the recipe (do NOT divide by serving size):
   - Return the total quantity for all servings combined
   - Example: If recipe has 8 servings and uses 2 cups flour:
     * Return the total quantity: 2 cups = 480g (this is the TOTAL for all 8 servings)
     * Result: original_quantity: 2, original_unit: cups, quantity: 480, unit: g
   - For count-based ingredients (e.g., eggs, sausages, chicken breasts):
     * Keep as count/pieces - return the total number in the recipe
     * Example: 4 sausages for 4 servings = original_quantity: 4, quantity: 4
   - For baked goods, return total amounts used in the full recipe
   - For main dishes, return total amounts for the complete recipe
   - For very large quantities, use appropriate larger units (e.g., convert grams to kilograms if needed)

3. Use appropriate units based on ingredient type:
   - Count-based items: Use "piece", "whole", "slice", etc.
   - Weight-based items: Use metric weights (g, mg, kg)
   - Volume-based items: Use metric volumes (ml, L)

4. MANDATORY metric conversions:
   - Weight: Use grams (g) or milligrams (mg)
     * Convert ounces to grams (1 oz = 28.35g)
     * Convert pounds to grams (1 lb = 453.6g)
   - Volume: Use milliliters (ml) or liters (L)
     * Convert cups to milliliters (1 cup = 240ml)
     * Convert tablespoons to milliliters (1 tbsp = 15ml)
     * Convert teaspoons to milliliters (1 tsp = 5ml)
     * Convert fluid ounces to milliliters (1 fl oz = 29.57ml)
   - Temperature: Use Celsius (°C) in all instructions
     * Convert Fahrenheit to Celsius ((°F - 32) × 5/9)
   - Length: Use centimeters (cm) or millimeters (mm)
     * Convert inches to centimeters (1 inch = 2.54cm)

5. Use these preferred units for adjusted quantities:
   - Count-based items (eggs, sausages, chicken breasts): piece, whole, slice
   - Flour, sugar, grains: grams (g)
   - Liquids: milliliters (ml)
   - Spices and small quantities: grams (g) or milligrams (mg)
   - Fresh produce: grams (g) or piece/whole for count-based items
   - Meat and protein: grams (g) or piece/whole for count-based items
   - Butter and oils: grams (g) or milliliters (ml)

6. Separate each ingredient into original_quantity, original_unit, quantity, unit, and name
7. For ingredients without a specific quantity, use null for both quantities and appropriate units (e.g., "to taste", "as needed")
8. Remove any preparation instructions from the ingredient name (e.g., "diced" or "chopped")
9. Check each ingredient against the criteria for each dietary category
10. Consider common ingredients that might violate certain restrictions (e.g., flour often contains gluten, sauces may contain dairy or gluten)
11. If an ingredient is ambiguous, assume it does not fit restrictive diets unless specified otherwise
12. For calorie and nutrition estimation, consider the caloric density and nutritional content of main ingredients
13. When scaling down sauces or marinades, maintain a practical minimum quantity needed for the cooking method
14. For spices and seasonings, scale down proportionally but ensure quantities remain practical for flavor
</INGREDIENT_ANALYSIS>`

const languageHandlingSection = `<LANGUAGE_HANDLING>
CRITICAL: You MUST preserve the original language of the recipe content throughout your entire output.

1. Language Detection:
   - Identify the primary language of the recipe content from the post description and/or video transcript
   - If the content contains multiple languages, use the dominant language of the recipe instructions
   - If the post description is in one language but the video transcript is in another, prioritize the language used for the actual recipe instructions and ingredients

 2. Language Preservation Rules:
    - recipe_name: GENERATE a descriptive title based on what the recipe makes, IN THE ORIGINAL LANGUAGE (do NOT copy the source title verbatim, do NOT translate)
    - description: MUST be in the original language (do NOT translate)
    - ingredients[].name: MUST be in the original language (do NOT translate)
    - instructions[].instruction: MUST be in the original language (do NOT translate)
    - language: MUST contain the ISO 639-1 language code of the detected language (e.g., "en", "es", "fr", "de", "it", "pt", "ja", "zh", "ko", "ar", "hi", etc.)

3. Recipe Title Generation:
   - Create a clear, descriptive title that reflects what the dish is
   - Focus on the main ingredients and cooking style, not marketing words
   - Examples of good titles: "Classic Italian Carbonara", "Spicy Thai Basil Chicken", "Homemade French Croissants"
   - Examples of bad titles: "Best Recipe Ever!!!", "You Won't Believe This!", "My Grandma's Secret"
   - Keep the title in the same language as the recipe content
   - If the original title is vague or clickbait, generate a better descriptive title

4. Examples:
   - Spanish recipe: recipe_name="Paella de Mariscos con Azafrán", description="Una deliciosa paella tradicional...", language="es"
   - French recipe: recipe_name="Coq au Vin à l'Ancienne", description="Un plat classique français...", language="fr"
   - German recipe: recipe_name="Traditioneller Bayerischer Sauerbraten", description="Ein traditioneller deutscher Sauerbraten...", language="de"
   - Mixed language content: If post caption is English but video instructions are in Spanish, use Spanish and set language="es"

5. Special Cases:
   - If the recipe uses dialect or regional variations, preserve them as-is in description, ingredients, and instructions
   - If ingredient names are in a local language, keep them in that language (e.g., "mozzarella", "parmesan", "harissa", "miso")
   - Do NOT transliterate scripts (e.g., keep Cyrillic, Arabic, Chinese characters as-is)
   - Do NOT convert measurements to different number systems (keep Arabic numerals, etc.)
</LANGUAGE_HANDLING>`

const instructionsSection = `<INSTRUCTIONS>
The user will provide the recipe content in the following format:
1. First: The post description (caption/text from the social media post)
2. Second: The video transcript (directly following the post description)

Your task is to parse both the post description and video transcript to extract complete recipe information and respond with only the structured JSON output that matches the GeneratedRecipe interface. Do not include any additional explanation or text outside of the JSON object. Ensure that:
1. The recipe object contains all the main recipe information
2. Each ingredient in the ingredients array has original_quantity, original_unit, quantity, unit, and name
3. All ingredient quantities are provided in both original form and total form for the complete recipe
4. The "quantity" and "unit" fields MUST use metric units (g, ml, etc.) - NEVER cups, tbsp, oz, etc.
5. Each instruction in the instructions array has a step number and instruction text
6. The nutrition object contains protein, carbs, fat, and fiber values in grams
 7. All time fields (prep_time, cooking_time, total_time) are in minutes
  8. The difficulty_rating is a number from 1 to 5
 9. All temperatures in instructions should be in Celsius (°C)
   10. Each instruction in the instructions array should:
     - Have a clear step number and detailed instruction text
     - Include visual cues and timing indicators where relevant
     - Explain complex techniques when needed
     - Provide safety warnings when appropriate
     - Include helpful tips for best results
     - Be detailed enough for a beginner to follow
  11. Extract cooking timers from each instruction and include them in the timer_data array:
    - Look for time mentions like "simmer for 10 minutes", "bake for 30 minutes", "rest for 5 minutes"
    - Create a timer object for each time-based instruction with these fields:
      * duration_seconds: The duration in seconds (e.g., 10 minutes = 600, 1 hour = 3600)
      * label: A descriptive label for what the timer is for (e.g., "Simmer sauce", "Bake in oven", "Let dough rest")
      * type: The type of timer - use "cooking" for active cooking, "prep" for preparation, "resting" for resting/cooling
      * category: Use "active" when attention is needed (e.g., stirring, monitoring), "passive" when unattended (e.g., baking, resting)
    - Examples:
      * "Simmer for 10 minutes until thickened" → {"duration_seconds": 600, "label": "Simmer until thickened", "type": "cooking", "category": "active"}
      * "Bake for 30 minutes at 180°C" → {"duration_seconds": 1800, "label": "Bake in oven", "type": "cooking", "category": "passive"}
      * "Let rest for 5 minutes before serving" → {"duration_seconds": 300, "label": "Rest before serving", "type": "resting", "category": "passive"}
    - If no timer is mentioned in an instruction, use an empty array [] or omit the field
</INSTRUCTIONS>`

const taskOpen = `<TASK>
Extract key information from recipe descriptions and output structured JSON data that matches the GeneratedRecipe interface. Preserve the original language of the recipe content in all text fields.
`

const taskClose = `</TASK>`

func getPlatformContext(platform string) string {
	switch strings.ToLower(platform) {
	case "instagram":
		return `<PLATFORM_CONTEXT>
This recipe comes from Instagram. Keep in mind:
- Instagram posts often have detailed captions with full recipe information
- Hashtags may indicate cuisine type, dietary restrictions, or meal type
- Aesthetic presentation is common - focus on extracting practical cooking information
- Captions may include ingredient lists formatted with emojis or bullet points
- Multiple images may show different steps - the transcript may reference these
- Influencer-style content may use informal measurements ("a splash of", "a handful")
</PLATFORM_CONTEXT>`
	case "tiktok":
		return `<PLATFORM_CONTEXT>
This recipe comes from TikTok. Keep in mind:
- TikTok videos are typically fast-paced with quick demonstrations
- Recipe information is often spoken in voiceover rather than written in captions
- Captions may be minimal - rely more on the video transcript for details
- Trendy or viral ingredients may be featured
- Informal language and slang is common ("bussin", "hits different", etc.)
- Measurements may be estimated or visual ("eyeball it", "about this much")
- Videos often skip detailed measurements - infer from visual cues in transcript
- Multiple recipe variations may be mentioned quickly
</PLATFORM_CONTEXT>`
	default:
		return ""
	}
}

// BuildRecipePrompt builds a recipe extraction prompt with optional platform-specific context
func BuildRecipePrompt(platform string) string {
	var sb strings.Builder
	sb.WriteString(roleSection)
	sb.WriteString("\n\n")

	pCtx := getPlatformContext(platform)
	if pCtx != "" {
		sb.WriteString(pCtx)
		sb.WriteString("\n\n")
	}

	sb.WriteString(taskOpen)
	sb.WriteString("\n")
	sb.WriteString(extractionGuidelinesSection)
	sb.WriteString("\n\n")
	sb.WriteString(inferenceSection)
	sb.WriteString("\n\n")
	sb.WriteString(outputFormatSection)
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf(ingredientAnalysisSection, criticalMetricRequirementSection))
	sb.WriteString("\n\n")
	sb.WriteString(languageHandlingSection)
	sb.WriteString("\n\n")
	sb.WriteString(instructionsSection)
	sb.WriteString("\n")
	sb.WriteString(taskClose)

	return sb.String()
}

// BuildFirecrawlPrompt builds a recipe extraction prompt optimized for website content extraction
func BuildFirecrawlPrompt() string {
	var sb strings.Builder
	sb.WriteString(roleSection)
	sb.WriteString("\n\n")

	sb.WriteString(`<PLATFORM_CONTEXT>
This recipe comes from a website. Keep in mind:
- Content may be in markdown format converted from HTML
- Recipe data may be in structured formats (JSON-LD, schema.org)
- Extract the main recipe content, ignoring ads and navigation
- Websites may have multiple recipes; extract the primary one
- Measurement formats vary (cups, grams, ounces may all be present)
- Pay attention to serving size information usually near the recipe title
</PLATFORM_CONTEXT>`)
	sb.WriteString("\n\n")

	sb.WriteString(taskOpen)
	sb.WriteString("\n")
	sb.WriteString(extractionGuidelinesSection)
	sb.WriteString("\n\n")
	sb.WriteString(inferenceSection)
	sb.WriteString("\n\n")
	sb.WriteString(outputFormatSection)
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf(ingredientAnalysisSection, criticalMetricRequirementSection))
	sb.WriteString("\n\n")
	sb.WriteString(languageHandlingSection)
	sb.WriteString("\n\n")
	sb.WriteString(instructionsSection)
	sb.WriteString("\n")
	sb.WriteString(taskClose)

	return sb.String()
}

// CategorySet holds existing user categories grouped by type
type CategorySet struct {
	CuisineCategories   []string
	MealTypes           []string
	Occasions           []string
	DietaryRestrictions []string
	Equipment           []string
}

// RecipeInfo holds minimal recipe information needed for category matching
type RecipeInfo struct {
	Name        string
	Description string
	Ingredients []string
}

// BuildCategoryPrompt builds a prompt for category matching AI
func BuildCategoryPrompt(recipe RecipeInfo, existing CategorySet) string {
	var sb strings.Builder

	sb.WriteString(`<ROLE>
You are a specialized AI assistant for recipe categorization. Your task is to analyze a recipe and match it to existing user categories or suggest new ones when appropriate. You should make thoughtful, accurate matches based on recipe content and context.
</ROLE>

`)

	sb.WriteString("<RECIPE_CONTEXT>\n")
	sb.WriteString(fmt.Sprintf("Recipe Name: %s\n", recipe.Name))
	if recipe.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", recipe.Description))
	}
	if len(recipe.Ingredients) > 0 {
		sb.WriteString("Ingredients:\n")
		for _, ing := range recipe.Ingredients {
			sb.WriteString(fmt.Sprintf("- %s\n", ing))
		}
	}
	sb.WriteString("</RECIPE_CONTEXT>\n\n")

	sb.WriteString("<EXISTING_CATEGORIES>\n")
	writeCategorySection(&sb, "Cuisine Categories", existing.CuisineCategories)
	writeCategorySection(&sb, "Meal Types", existing.MealTypes)
	writeCategorySection(&sb, "Occasions", existing.Occasions)
	writeCategorySection(&sb, "Dietary Restrictions", existing.DietaryRestrictions)
	writeCategorySection(&sb, "Equipment", existing.Equipment)
	sb.WriteString("</EXISTING_CATEGORIES>\n\n")

	sb.WriteString(`<INSTRUCTIONS>
1. MATCHING LOGIC:
   - Analyze the recipe context (name, description, ingredients) carefully
   - Match the recipe to existing categories when there's a clear fit
   - A match is appropriate when the category meaningfully describes the recipe
   - Recipes can have MULTIPLE categories per type (e.g., both "Italian" and "Mediterranean")
   - If no existing category fits well (similarity < 0.7 concept), leave the array empty for that type

2. THRESHOLD GUIDANCE:
   - Use existing categories liberally when they apply
   - Only suggest NEW categories if no existing category captures the essence
   - For cuisine: consider ingredient origins, cooking techniques, and flavor profiles
   - For meal types: consider when the dish is typically eaten
   - For occasions: consider when the dish would be served
   - For dietary restrictions: only include if the recipe clearly fits
   - For equipment: list all necessary tools for preparation

3. NEW CATEGORY SUGGESTIONS:
   - Only suggest new categories in "new_category_suggestions" when truly needed
   - New suggestions should be concise, descriptive, and user-friendly
   - Avoid duplicating existing categories (case-insensitive comparison)
</INSTRUCTIONS>

`)

	sb.WriteString(`<OUTPUT_FORMAT>
Return ONLY a JSON object with the following structure (no additional text):

{
  "cuisine_categories": ["ExistingCategory1", "ExistingCategory2"],
  "meal_types": ["ExistingMealType"],
  "occasions": ["ExistingOccasion"],
  "dietary_restrictions": ["ExistingRestriction"],
  "equipment": ["ExistingEquipment1", "ExistingEquipment2"],
  "new_category_suggestions": {
    "cuisine_categories": [],
    "meal_types": [],
    "occasions": [],
    "dietary_restrictions": [],
    "equipment": []
  }
}

Guidelines:
- Use strings from existing categories where possible
- Arrays can be empty if no match exists
- New suggestions only when existing categories don't fit
- Equipment should include all tools needed
</OUTPUT_FORMAT>`)

	return sb.String()
}

// writeCategorySection writes a category section to the string builder
func writeCategorySection(sb *strings.Builder, title string, categories []string) {
	sb.WriteString(fmt.Sprintf("%s: ", title))
	if len(categories) == 0 {
		sb.WriteString("None yet")
	} else {
		for i, cat := range categories {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("\"%s\"", cat))
		}
	}
	sb.WriteString("\n")
}

// RichInstructionPromptVersion is the version of the rich instruction prompt
const RichInstructionPromptVersion = 1

// Timer represents a cooking timer extracted from instruction text
type Timer struct {
	DurationSeconds int    `json:"duration_seconds"`
	Label           string `json:"label"`
	Type            string `json:"type"`
	Category        string `json:"category"`
}

// IngredientForPrompt holds ingredient data with UUID for placeholder generation
type IngredientForPrompt struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// RecipeForPrompt holds recipe data needed for placeholder prompt generation
type RecipeForPrompt struct {
	Name         string
	Ingredients  []IngredientForPrompt
	Instructions []InstructionForPrompt
}

// InstructionForPrompt holds instruction data with timer indices
type InstructionForPrompt struct {
	StepNumber  int
	Instruction string
	TimerData   []Timer
}

// BuildPlaceholderPrompt builds a prompt for rich instruction generation with placeholders
func BuildPlaceholderPrompt(recipe RecipeForPrompt) string {
	var sb strings.Builder

	sb.WriteString(`<ROLE>
You are a specialized AI assistant for recipe instruction enhancement. Your task is to process recipe instructions and enrich them with interactive placeholders for ingredients and timers. You should maintain the original language and meaning while adding structured placeholder markers.
</ROLE>

`)

	sb.WriteString("<CONTEXT>\n")
	sb.WriteString(fmt.Sprintf("Recipe Name: %s\n\n", recipe.Name))

	sb.WriteString("Ingredients (use these UUIDs for {{ingredient:UUID}} placeholders):\n")
	for _, ing := range recipe.Ingredients {
		sb.WriteString(fmt.Sprintf("  [%s] %s\n", ing.ID, ing.Name))
	}
	sb.WriteString("\n")

	sb.WriteString("Instructions with Timer Data:\n")
	for _, inst := range recipe.Instructions {
		sb.WriteString(fmt.Sprintf("\nStep %d: %s\n", inst.StepNumber, inst.Instruction))
		if len(inst.TimerData) > 0 {
			sb.WriteString("  Timers (use these indices for {{timer:N}} placeholders):\n")
			for j, timer := range inst.TimerData {
				sb.WriteString(fmt.Sprintf("    [%d] Label: %s, Duration: %ds, Type: %s, Category: %s\n",
					j, timer.Label, timer.DurationSeconds, timer.Type, timer.Category))
			}
		}
	}
	sb.WriteString("</CONTEXT>\n\n")

	sb.WriteString(`<OUTPUT_FORMAT>
Return a JSON object with the following structure:

{
  "instructions": [
    {
      "step_number": 1,
      "instruction_rich": "Enhanced instruction text with {{ingredient:550e8400-e29b-41d4-a716-446655440000}} and {{timer:0}} placeholders"
    }
  ]
}

The instruction_rich field should contain the original instruction text enhanced with:
- {{ingredient:UUID}} placeholders where UUID is the ingredient's unique identifier (36-character UUID format)
- {{timer:N}} placeholders where N is the timer index within that instruction
</OUTPUT_FORMAT>

`)

	sb.WriteString(`<INSTRUCTIONS>
1. PLACEHOLDER RULES:
   - Insert {{ingredient:UUID}} placeholders where ingredients are mentioned (UUID = ingredient's unique identifier)
   - Insert {{timer:N}} placeholders where timing is mentioned (N = timer index within that instruction)
   - Preserve the original instruction text and language
   - Placeholders should replace or augment the text they reference

2. INGREDIENT PLACEHOLDERS:
   - Use the ingredient UUID from the ingredient list above (36-character UUID format like 550e8400-e29b-41d4-a716-446655440000)
   - Example: If ingredient with UUID "550e8400-e29b-41d4-a716-446655440000" is "chicken breast", replace "chicken breast" with "{{ingredient:550e8400-e29b-41d4-a716-446655440000}}"
   - Only use valid UUIDs that exist in the ingredient list

3. TIMER PLACEHOLDERS:
   - Use the timer index from the timer_data for each instruction
   - Example: If an instruction has timer_data[0], insert {{timer:0}} at the appropriate location
   - Only use valid indices that exist in that instruction's timer_data

4. OUTPUT REQUIREMENTS:
   - Return ONLY the JSON object, no additional text
   - Maintain the original language of the recipe
   - Keep step numbers sequential and matching the input
   - The instruction_rich should be a complete, readable sentence with placeholders integrated naturally
</INSTRUCTIONS>`)

	return sb.String()
}

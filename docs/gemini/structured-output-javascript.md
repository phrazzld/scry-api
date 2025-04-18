# Generate structured output with the Gemini API

Gemini generates unstructured text by default, but some applications require structured text. For these use cases, you can constrain Gemini to respond with JSON, a structured data format suitable for automated processing. You can also constrain the model to respond with one of the options specified in an enum.

Here are a few use cases that might require structured output from the model:

- Build a database of companies by pulling company information out of newspaper articles.
- Pull standardized information out of resumes.
- Extract ingredients from recipes and display a link to a grocery website for each ingredient.

In your prompt, you can ask Gemini to produce JSON-formatted output, but note that the model is not guaranteed to produce JSON and nothing but JSON. For a more deterministic response, you can pass a specific JSON schema in a responseSchema field so that Gemini always responds with an expected structure. To learn more about working with schemas, see More about JSON schemas.

This guide shows you how to generate JSON using the generateContent method through the SDK of your choice or using the REST API directly. The examples show text-only input, although Gemini can also produce JSON responses to multimodal requests that include images, videos, and audio.

## Generate JSON

When the model is configured to output JSON, it responds to any prompt with JSON-formatted output.

You can control the structure of the JSON response by supplying a schema. There are two ways to supply a schema to the model:

1. As text in the prompt
2. As a structured schema supplied through model configuration

### Supply a schema as text in the prompt

The following example prompts the model to return cookie recipes in a specific JSON format.

Since the model gets the format specification from text in the prompt, you may have some flexibility in how you represent the specification. Any reasonable format for representing a JSON schema may work.

```javascript
import { GoogleGenAI } from "@google/genai";

const ai = new GoogleGenAI({ apiKey: "GEMINI_API_KEY" });

async function main() {
    const prompt = `List a few popular cookie recipes using this JSON schema:

    Recipe = {'recipeName': string}
    Return: Array<Recipe>`;

    const response = await ai.models.generateContent({
        model: "gemini-2.0-flash",
        contents: prompt,
    });
    console.log(response.text);
}

main();
```

The output might look like this:

```json
[{"recipeName": "Chocolate Chip Cookies"}, {"recipeName": "Oatmeal Raisin Cookies"}, {"recipeName": "Snickerdoodles"}, {"recipeName": "Sugar Cookies"}, {"recipeName": "Peanut Butter Cookies"}]
```

### Supply a schema through model configuration

The following example does the following:

1. Instantiates a model configured through a schema to respond with JSON.
2. Prompts the model to return cookie recipes.

This more formal method for declaring the JSON schema gives you more precise control than relying just on text in the prompt.

> Important: When you're working with JSON schemas in the Gemini API, the order of properties matters. For more information, see Property ordering.

```javascript
import { GoogleGenAI, Type } from "@google/genai";

const ai = new GoogleGenAI({ apiKey: "GEMINI_API_KEY" });

async function main() {
    const response = await ai.models.generateContent({
        model: 'gemini-2.0-flash',
        contents: 'List 3 popular cookie recipes.',
        config: {
            responseMimeType: 'application/json',
            responseSchema: {
                type: Type.ARRAY,
                items: {
                    type: Type.OBJECT,
                    properties: {
                        'recipeName': {
                            type: Type.STRING,
                            description: 'Name of the recipe',
                            nullable: false,
                        },
                    },
                    required: ['recipeName'],
                },
            },
        },
    });

    console.debug(response.text);
}

main();
```

The output might look like this:

```json
[{"recipeName": "Chocolate Chip Cookies"}, {"recipeName": "Oatmeal Raisin Cookies"}, {"recipeName": "Snickerdoodles"}, {"recipeName": "Sugar Cookies"}, {"recipeName": "Peanut Butter Cookies"}]
```

## More about JSON schemas

When you configure the model to return a JSON response, you can use a Schema object to define the shape of the JSON data. The Schema represents a select subset of the OpenAPI 3.0 Schema object.

Here's a pseudo-JSON representation of all the Schema fields:

```json
{
  "type": enum (Type),
  "format": string,
  "description": string,
  "nullable": boolean,
  "enum": [
    string
  ],
  "maxItems": string,
  "minItems": string,
  "properties": {
    string: {
      object (Schema)
    },
    ...
  },
  "required": [
    string
  ],
  "propertyOrdering": [
    string
  ],
  "items": {
    object (Schema)
  }
}
```

The Type of the schema must be one of the OpenAPI Data Types. Only a subset of fields is valid for each Type. The following list maps each Type to valid fields for that type:

- string -> enum, format
- integer -> format
- number -> format
- boolean
- array -> minItems, maxItems, items
- object -> properties, required, propertyOrdering, nullable

Here are some example schemas showing valid type-and-field combinations:

```json
{ "type": "string", "enum": ["a", "b", "c"] }

{ "type": "string", "format": "date-time" }

{ "type": "integer", "format": "int64" }

{ "type": "number", "format": "double" }

{ "type": "boolean" }

{ "type": "array", "minItems": 3, "maxItems": 3, "items": { "type": ... } }

{ "type": "object",
  "properties": {
    "a": { "type": ... },
    "b": { "type": ... },
    "c": { "type": ... }
  },
  "nullable": true,
  "required": ["c"],
  "propertyOrdering": ["c", "b", "a"]
}
```

For complete documentation of the Schema fields as they're used in the Gemini API, see the Schema reference.

## Property ordering

When you're working with JSON schemas in the Gemini API, the order of properties is important. By default, the API orders properties alphabetically and does not preserve the order in which the properties are defined (although the Google Gen AI SDKs may preserve this order). If you're providing examples to the model with a schema configured, and the property ordering of the examples is not consistent with the property ordering of the schema, the output could be rambling or unexpected.

To ensure a consistent, predictable ordering of properties, you can use the optional propertyOrdering[] field.

```json
"propertyOrdering": ["recipe_name", "ingredients"]
```

propertyOrdering[] – not a standard field in the OpenAPI specification – is an array of strings used to determine the order of properties in the response. By specifying the order of properties and then providing examples with properties in that same order, you can potentially improve the quality of results.

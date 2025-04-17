# Model variants

The Gemini API offers different models that are optimized for specific use cases. Here's a brief overview of Gemini variants that are available:

| Model variant | Input(s) | Output | Optimized for |
|---------------|----------|--------|---------------|
| **Gemini 2.5 Pro Experimental** (gemini-2.5-pro-exp-03-25) | Audio, images, videos, and text | Text | Enhanced thinking and reasoning, multimodal understanding, advanced coding, and more |
| **Gemini 2.0 Flash** (gemini-2.0-flash) | Audio, images, videos, and text | Text, images (experimental), and audio (coming soon) | Next generation features, speed, thinking, realtime streaming, and multimodal generation |
| **Gemini 2.0 Flash-Lite** (gemini-2.0-flash-lite) | Audio, images, videos, and text | Text | Cost efficiency and low latency |
| **Gemini 1.5 Flash** (gemini-1.5-flash) | Audio, images, videos, and text | Text | Fast and versatile performance across a diverse variety of tasks |
| **Gemini 1.5 Flash-8B** (gemini-1.5-flash-8b) | Audio, images, videos, and text | Text | High volume and lower intelligence tasks |
| **Gemini 1.5 Pro** (gemini-1.5-pro) | Audio, images, videos, and text | Text | Complex reasoning tasks requiring more intelligence |
| **Gemini Embedding** (gemini-embedding-exp) | Text | Text embeddings | Measuring the relatedness of text strings |
| **Imagen 3** (imagen-3.0-generate-002) | Text | Images | Our most advanced image generation model |

You can view the rate limits for each model on the rate limits page.

## Gemini 2.5 Pro Experimental

Gemini 2.5 Pro Experimental is our state-of-the-art thinking model, capable of reasoning over complex problems in code, math, and STEM, as well as analyzing large datasets, codebases, and documents using long context.

[Try in Google AI Studio](https://aistudio.google.com)

### Model details

| Property | Description |
|----------|-------------|
| **Model code** | gemini-2.5-pro-exp-03-25 |
| **Supported data types** | |
| Inputs | Audio, images, video, and text |
| Output | Text |
| **Token limits[*]** | |
| Input token limit | 1,048,576 |
| Output token limit | 65,536 |
| **Capabilities** | |
| Structured outputs | Supported |
| Caching | Not supported |
| Tuning | Not supported |
| Function calling | Supported |
| Code execution | Supported |
| Search grounding | Supported |
| Image generation | Not supported |
| Native tool use | Supported |
| Audio generation | Not supported |
| Live API | Not supported |
| Thinking | Supported |
| **Versions** | Read the model version patterns for more details. |
| Experimental | gemini-2.5-pro-exp-03-25 |
| **Latest update** | March 2025 |
| **Knowledge cutoff** | January 2025 |

## Gemini 2.0 Flash

Gemini 2.0 Flash delivers next-gen features and improved capabilities, including superior speed, native tool use, multimodal generation, and a 1M token context window.

[Try in Google AI Studio](https://aistudio.google.com)

### Model details

| Property | Description |
|----------|-------------|
| **Model code** | models/gemini-2.0-flash |
| **Supported data types** | |
| Inputs | Audio, images, video, and text |
| Output | Text, images (experimental), and audio(coming soon) |
| **Token limits[*]** | |
| Input token limit | 1,048,576 |
| Output token limit | 8,192 |
| **Capabilities** | |
| Structured outputs | Supported |
| Caching | Coming soon |
| Tuning | Not supported |
| Function calling | Supported |
| Code execution | Supported |
| Search | Supported |
| Image generation | Experimental |
| Native tool use | Supported |
| Audio generation | Coming soon |
| Live API | Experimental |
| Thinking | Experimental |
| **Versions** | Read the model version patterns for more details. |
| Latest | gemini-2.0-flash |
| Stable | gemini-2.0-flash-001 |
| Experimental | gemini-2.0-flash-exp and gemini-2.0-flash-exp-image-generation point to the same underlying model |
| Experimental | gemini-2.0-flash-thinking-exp-01-21 |
| **Latest update** | February 2025 |
| **Knowledge cutoff** | August 2024 |

## Gemini 2.0 Flash-Lite

A Gemini 2.0 Flash model optimized for cost efficiency and low latency.

[Try in Google AI Studio](https://aistudio.google.com)

### Model details

| Property | Description |
|----------|-------------|
| **Model code** | models/gemini-2.0-flash-lite |
| **Supported data types** | |
| Inputs | Audio, images, video, and text |
| Output | Text |
| **Token limits[*]** | |
| Input token limit | 1,048,576 |
| Output token limit | 8,192 |
| **Capabilities** | |
| Structured outputs | Supported |
| Caching | Not supported |
| Tuning | Not supported |
| Function calling | Not supported |
| Code execution | Not supported |
| Search | Not supported |
| Image generation | Not supported |
| Native tool use | Not supported |
| Audio generation | Not supported |
| Live API | Not supported |
| **Versions** | Read the model version patterns for more details. |
| Latest | gemini-2.0-flash-lite |
| Stable | gemini-2.0-flash-lite-001 |
| **Latest update** | February 2025 |
| **Knowledge cutoff** | August 2024 |

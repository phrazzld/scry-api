# Gemini API quickstart

This quickstart shows you how to install your SDK of choice and then make your first Gemini API request.

## Install the Gemini API library

> Note: We're rolling out a new set of Gemini API libraries, the Google Gen AI SDK.

Using Node.js v18+, install the Google Gen AI SDK for TypeScript and JavaScript using the following npm command:

```javascript
npm install @google/genai
```

## Make your first request

1. Get a Gemini API key in Google AI Studio

2. Use the generateContent method to send a request to the Gemini API.

```javascript
import { GoogleGenAI } from "@google/genai";

const ai = new GoogleGenAI({ apiKey: "YOUR_API_KEY" });

async function main() {
  const response = await ai.models.generateContent({
    model: "gemini-2.0-flash",
    contents: "Explain how AI works in a few words",
  });
  console.log(response.text);
}

main();
```

import { ChatMessageType, ModalList, AgentList, useSettings } from "../store/store";

// Base API URL for the backend
const BASE_API_URL = "http://127.0.0.1:8080/v1";

// Map agent types to their API endpoints
const AGENT_ENDPOINTS: Record<AgentList, string> = {
    "general": "/chat/completions",
    "study-materials": "/chat/resources",
    "academic-advisor": "/chat/advisor",
    "notifications": "/chat/notifications"
};

const IMAGE_GENERATION_API_URL = "https://api.openai.com/v1/images/generations";

export async function fetchResults(
    messages: Omit<ChatMessageType, "id" | "type">[],
    model: string,
    signal: AbortSignal,
    onData: (data: string) => void,
    onCompletion: () => void
) {
    try {
        // Get currently selected agent
        const selectedAgent = useSettings.getState().settings.selectedAgent;

        // Construct the appropriate endpoint based on agent type
        const endpoint = AGENT_ENDPOINTS[selectedAgent];
        const apiUrl = `${BASE_API_URL}${endpoint}`;

        // Common request parameters for all agents
        const requestBody: any = {
            model: model,
            temperature: 0.7,
            stream: true,
            messages: messages
        };

        const response = await fetch(apiUrl, {
            method: "POST",
            signal: signal,
            headers: {
                "Content-Type": "application/json",
                "Accept": "text/event-stream",
                "Authorization": `Bearer ${localStorage.getItem("access_token")}` // Using token instead of API key
            },
            body: JSON.stringify(requestBody)
        });

        if (!response.ok) {
            const errorData = await response.json().catch(() => ({}));
            throw new Error(errorData.message || `Error fetching results: ${response.status}`);
        }

        const reader = response.body?.getReader();
        if (!reader) {
            throw new Error("Response body cannot be read");
        }

        while (true) {
            const { done, value } = await reader.read();

            if (done) {
                onCompletion();
                break;
            }

            const chunk = new TextDecoder("utf-8").decode(value, { stream: true });
            const chunks = chunk.split("\n").filter(x => x !== "");

            chunks.forEach((chunk: string) => {
                if (chunk === "data: [DONE]") {
                    return;
                }

                if (!chunk.startsWith("data: ")) return;

                chunk = chunk.replace("data: ", "");
                try {
                    const data = JSON.parse(chunk);

                    // Handle different response formats based on agent type
                    let content: string;

                    if (selectedAgent === "study-materials" && data.resource) {
                        content = data.resource.content || data.choices?.[0]?.delta?.content || "";
                    } else if (selectedAgent === "notifications" && data.notification) {
                        content = data.notification.message || data.choices?.[0]?.delta?.content || "";
                    } else {
                        // Default format for general and academic-advisor
                        content = data.choices?.[0]?.delta?.content || "";
                    }

                    if (content) {
                        onData(content);
                    }

                    if (data.choices?.[0]?.finish_reason === "stop") return;
                } catch (e) {
                    console.error("Error parsing chunk:", e);
                }
            });
        }
    } catch (error) {
        if (error instanceof DOMException || error instanceof Error) {
            throw new Error(error.message);
        } else {
            throw new Error("Unknown error occurred");
        }
    }
}

export type ImageSize =
  | "256x256"
  | "512x512"
  | "1024x1024"
  | "1280x720"
  | "1920x1080"
  | "1024x1024"
  | "1792x1024"
  | "1024x1792";

export type IMAGE_RESPONSE = {
  created_at: string;
  data: IMAGE[];
};
export type IMAGE = {
  url: string;
};
export type DallEImageModel = Extract<ModalList, "dall-e-2" | "dall-e-3">;

export async function generateImage(
  prompt: string,
  size: ImageSize,
  numberOfImages: number
) {
  const selectedModal = useSettings.getState().settings.selectedModal;

  const response = await fetch(IMAGE_GENERATION_API_URL, {
    method: `POST`,
    // signal: signal,
    headers: {
      "content-type": `application/json`,
      accept: `text/event-stream`,
      Authorization: `Bearer ${localStorage.getItem("apikey")}`,
    },
    body: JSON.stringify({
      model: selectedModal,
      prompt: prompt,
      n: numberOfImages,
      size: useSettings.getState().settings.dalleImageSize[
        selectedModal as DallEImageModel
      ],
    }),
  });
  const body: IMAGE_RESPONSE = await response.json();
  return body;
}

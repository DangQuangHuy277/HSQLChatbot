import {create} from "zustand";
import {persist} from "zustand/middleware";
import {createWithEqualityFn} from "zustand/traditional";
import {shallow} from "zustand/shallow";
import {produce} from "immer";
import moment from "moment";
import {ImageSize} from "../services/chatService";

const modalsList = [
    "gpt-3.5-turbo",
    "gpt-3.5-turbo-1106",
    "gpt-3.5-turbo-16k-0613",
    "gpt-3.5-turbo-16k",
    "gpt-3.5-turbo-0613",
    "gpt-4",
    "gpt-4-0613",
    "gpt-4-0314",
    "gpt-4-1106-preview",
    "dall-e-3",
    "dall-e-2",
] as const;

const BASE_API_URL = "http://127.0.0.1:8080/api/v1";

const agentsList = [
    "general",
    "article",
    "email_support",
]

export interface ChatMessageType {
    role: "user" | "bot";
    content: string;
    id: string;
}

export interface SystemMessageType {
    message: string;
    useForAllChats: boolean;
}

export interface ModalPermissionType {
    id: string;
    object: string;
    created: number;
    allow_create_engine: boolean;
    allow_sampling: boolean;
    allow_logprobs: boolean;
    allow_search_indices: boolean;
    allow_view: boolean;
    allow_fine_tuning: boolean;
    organization: string;
    group: null;
    is_blocking: boolean;
}

export interface ModalType {
    id: string;
    object: string;
    created: number;
    owned_by: string;
    permission: ModalPermissionType[];
    root: string;
    parent: null;
}

export type Theme = "light" | "dark";

export interface ThemeType {
    theme: Theme;
    setTheme: (theme: Theme) => void;
}

export type ModalList = (typeof modalsList)[number];

export type AgentList = (typeof agentsList)[number];

export interface SettingsType {
    settings: {
        sendChatHistory: boolean;
        systemMessage: string;
        useSystemMessageForAllChats: boolean;
        selectedModal: ModalList;
        dalleImageSize: { "dall-e-2": ImageSize; "dall-e-3": ImageSize };
        selectedAgent: AgentList;
    };
    modalsList: readonly string[];
    isSystemMessageModalVisible: boolean;
    isModalVisible: boolean;
    isConfirmLogoutVisible: boolean;
    setConfirmLogout: (value: boolean) => void;
    setSystemMessage: (value: SystemMessageType) => void;
    setSystemMessageModalVisible: (value: boolean) => void;
    setSendChatHistory: (value: boolean) => void;
    setModalVisible: (value: boolean) => void;
    setModalsList: (value: string[]) => void;
    setModal: (value: ModalList) => void;
    setDalleImageSize: (value: ImageSize, type: "dall-e-2" | "dall-e-3") => void;
    setAgent: (value: AgentList) => void;
}

export interface ChatType {
    chats: ChatMessageType[];
    chatHistory: string[]; // contain all conversation ids
    conversations: any[]; // contain all conversations info
    currentConversation: string;
    initChatHistory: () => void;
    addChat: (chat: ChatMessageType, index?: number) => void;
    editChatMessage: (chat: string, updateIndex: number) => void;
    addNewChat: () => void;
    saveChats: () => void;
    viewSelectedChat: (chatId: string) => void;
    resetChatAt: (index: number) => void;
    handleDeleteChats: (chatid: string) => void;
    editChatsTitle: (id: string, title: string) => void;
    clearAllChats: () => void;
}

export interface UserType {
    name: string;
    email: string;
    avatar: string;
}

export interface AuthType {
    token: string;
    apikey: string;
    setToken: (token: string) => void;
    setUser: (user: { name: string; email: string; avatar: string }) => void;
    setApiKey: (apikey: string) => void;
    user: UserType;
    accessToken: string;
    logout: () => void;
    login: (token: string) => void;
}

// (set, get) => ({}) is createState callback of create, set and get is internal function of zustand
const useChat = create<ChatType>((set, get) => ({
    chats: [],
    chatHistory: [],
    conversations: [],
    currentConversation: "",

    initChatHistory: async () => {
        const response = await fetch(`${BASE_API_URL}/conversations`, {
            headers: {
               Authorization: `Bearer ${useAuth.getState().token}`,
            }
        });
        const data = await response.json();
        set({ chatHistory: data.map((c: any) => c.id) });
        set({ conversations: data });
    },
    addChat: (chat, index) => {
        fetch(`${BASE_API_URL}/messages`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                Authorization: `Bearer ${useAuth.getState().token}`,
            },
            body: JSON.stringify({...chat, conversation_id: get().currentConversation}),
        })
        set(
            produce((state: ChatType) => {
                if (index || index === 0) state.chats[index] = chat;
                else {
                    state.chats.push(chat);
                }
            })
        );
        // if (chat.role === "bot" && chat.content) {
        //     get().saveChats();
        // }
    },
    editChatMessage: (chat, updateIndex) => {
        set(
            produce((state: ChatType) => {
                state.chats[updateIndex].content = chat;
            })
        );
    },
    addNewChat: () => {
        if (get().chats.length === 0) return;
        set(
            produce((state: ChatType) => {
                state.chats = [];
            })
        );
    },

    saveChats: async () => { // no used
        let chat_id = get().chats[0].id;
        const response = await fetch(`${BASE_API_URL}/chats`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                Authorization: `Bearer ${useAuth.getState().token}`,
            },
            body: JSON.stringify({
                id: chat_id,
                createdAt: new Date().toISOString(),
                chats: get().chats,
                title: get().chats[0].content,
                isTitleEdited: false,
                agentType: useSettings.getState().settings.selectedAgent,
            }),
        });
        const data = await response.json();
        if (response.status !== 201) {
            console.error("Failed to save chat history", data);
            return;
        }
        const chatHistory = await fetch(`${BASE_API_URL}/chats`, {
            headers: {
                Authorization: `Bearer ${useAuth.getState().token}`,
            },
        });
        const chatHistoryData = await chatHistory.json();
        set(
            produce((state: ChatType) => {
                state.chatHistory = chatHistoryData;
            })
        );
    },
    viewSelectedChat: async (conversationId) => {
        const response = await fetch(`${BASE_API_URL}/conversations/${conversationId}`, {
            headers: {
                Authorization: `Bearer ${useAuth.getState().token}`,
            },
        });
        const data = await response.json();
        if (response.status !== 200) {
            console.error("Failed to get chat", data);
            return;
        }
        set(
            produce((state: ChatType) => {
                state.chats = data.chats ?? [];
                state.currentConversation = conversationId;

                // Set the agent type for this chat
                if (data.agentType) {
                    // Switch to the correct agent for this chat
                    useSettings.getState().setAgent(data.agentType);
                }
            })
        );
    },
    resetChatAt: (index) => {
        set(
            produce((state: ChatType) => {
                state.chats[index].content = "";
            })
        );
    },
    handleDeleteChats: async (conversationId) => {
        try {
            const response = await fetch(`${BASE_API_URL}/conversations/${conversationId}`, {
                method: "DELETE",
                headers: {
                    Authorization: `Bearer ${useAuth.getState().token}`,
                },
            });

            if (!response.ok) {
                console.error("Failed to delete chat", await response.json());
                return;
            }

            set(
                produce((state: ChatType) => {
                    state.chatHistory = state.chatHistory.filter((id) => id !== conversationId);
                    state.chats = [];
                    state.currentConversation = "";
                })
            );
        } catch (error) {
            console.error("Error deleting chat", error);
        }
    },
    editChatsTitle: async (id, title) => {
        try {
            const response = await fetch(`${BASE_API_URL}/conversations/${id}`, {
                method: "PATCH",
                headers: {
                    "Content-Type": "application/json",
                    Authorization: `Bearer ${useAuth.getState().token}`,
                },
                body: JSON.stringify({ title, isTitleEdited: true }),
            });

            if (!response.ok) {
                console.error("Failed to edit chat title", await response.json());
                return;
            }
        } catch (error) {
            console.error("Error editing chat title", error);
        }
    },
    clearAllChats: async () => {
        try {
            const response = await fetch(`${BASE_API_URL}/conversations`, {
                method: "DELETE",
                headers: {
                    Authorization: `Bearer ${useAuth.getState().token}`,
                },
            });

            if (!response.ok) {
                console.error("Failed to clear all chats", await response.json());
                return;
            }

            set(
                produce((state: ChatType) => {
                    state.chats = [];
                    state.chatHistory = [];
                    state.currentConversation = "";
                })
            );
        } catch (error) {
            console.error("Error clearing all chats", error);
        }
    },
}));

// create call return createImpl function -> Call it with persist middleware (which return newImpl) a.k.a pass newImpl as createState callback of createImpl
// State: a persist localStorage   n state with some along function tools
// Return
const useAuth = create<AuthType>()(
    // persist receive config(which is (set,get,api) => {} callback) and baseOptions and return newImpl callback
    //
    persist(
        (set) => ({
            token: localStorage.getItem("token") || "",
            apikey: localStorage.getItem("apikey") || "",
            user: {
                name: "Your name?",
                email: "",
                avatar: "/imgs/default-avatar.jpg",
            },
            accessToken: localStorage.getItem("access_token") || "",
            setToken: (token) => {
                set(
                    produce((state) => {
                        state.token = token;
                    })
                );
            },
            setUser: (user) => {
                set(
                    produce((state) => {
                        state.user = user;
                    })
                );
            },
            setApiKey: (apikey) => {
                set(
                    produce((state) => {
                        state.apikey = apikey;
                    })
                );
                localStorage.setItem("apikey", apikey);
            },
            logout: () => {
                set(
                    produce((state) => {
                        state.accessToken = "";
                    })
                );
                localStorage.removeItem("access_token");
            },
            login: (token) => {
                set(
                    produce((state) => {
                        state.accessToken = token;
                    })
                );
                localStorage.setItem("access_token", token);
            }
        }),
        {
            name: "auth",
        }
    )
);

const useSettings = createWithEqualityFn<SettingsType>()(
    persist(
        (set) => ({
            settings: {
                sendChatHistory: false,
                systemMessage: "",
                useSystemMessageForAllChats: false,
                selectedModal: "gpt-3.5-turbo",
                dalleImageSize: {"dall-e-2": "256x256", "dall-e-3": "1024x1024"},
                selectedAgent: "general",
            },
            modalsList: modalsList,
            isSystemMessageModalVisible: false,
            isModalVisible: false,
            isConfirmLogoutVisible: false,
            setConfirmLogout: (value) => {
                set(
                    produce((state: SettingsType) => {
                        state.isConfirmLogoutVisible = value;
                    })
                );
            },
            setSystemMessage: (value) => {
                set(
                    produce((state: SettingsType) => {
                        state.settings.systemMessage = value.message;
                        state.settings.useSystemMessageForAllChats = value.useForAllChats;
                    })
                );
            },
            setSystemMessageModalVisible: (value) => {
                set(
                    produce((state: SettingsType) => {
                        state.isSystemMessageModalVisible = value;
                    })
                );
            },
            setSendChatHistory: (value) => {
                set(
                    produce((state: SettingsType) => {
                        state.settings.sendChatHistory = value;
                    })
                );
            },
            setModal: (value) => {
                set(
                    produce((state: SettingsType) => {
                        state.settings.selectedModal = value;
                    })
                );
            },
            setModalVisible: (value) => {
                set(
                    produce((state: SettingsType) => {
                        state.isModalVisible = value;
                    })
                );
            },
            setModalsList: (value) => {
                set(
                    produce((state: SettingsType) => {
                        state.modalsList = value;
                    })
                );
            },
            setDalleImageSize: (value, type) => {
                set(
                    produce((state: SettingsType) => {
                        state.settings.dalleImageSize[type] = value;
                    })
                );
            },
            setAgent: (value) => {
                set(
                    produce((state: SettingsType) => {
                        state.settings.selectedAgent = value;
                    })
                );
            }
        }),
        {
            name: "settings",
            version: 1,
            partialize: (state: SettingsType) => ({settings: state.settings}),
            migrate: (persistedState: unknown, version: number) => {
                if (version === 0) {
                    (persistedState as SettingsType["settings"]).dalleImageSize = {
                        "dall-e-2": "256x256",
                        "dall-e-3": "1024x1024",
                    };
                }

                return persistedState as SettingsType;
            },
        }
    ),
    shallow
);

const useTheme = create<ThemeType>()(
    persist(
        (set) => ({
            theme: "dark",
            setTheme: (theme) => {
                set(
                    produce((state) => {
                        state.theme = theme;
                    })
                );
            },
        }),
        {
            name: "theme",
        }
    )
);

export const months = [
    "January",
    "February",
    "March",
    "April",
    "May",
    "June",
    "July",
    "August",
    "September",
    "October",
    "November",
    "December",
];
export const priority = [
    "Today",
    "Previous 7 Days",
    "Previous 30 Days",
    "This month",
].concat(months);

export const selectChatsHistory = (state: ChatType) => {
    const sortedData: Record<string,
        { title: string; id: string; month: string; month_id: number, agentType: AgentList }[]> = {};
    state.conversations.forEach( ({title, id, created_at, agentType}) => {
        const myDate = moment(created_at, "YYYY-MM-DD");
        const currentDate = moment();
        const month = myDate.toDate().getMonth();

        const data = {
            title,
            id,
            month: months[month],
            month_id: month,
            agentType: agentType || "general" // Default to general if not specified
        };

        if (myDate.isSame(currentDate.format("YYYY-MM-DD"))) {
            if (!sortedData.hasOwnProperty("Today")) {
                sortedData["Today"] = [];
            }
            sortedData["Today"].push(data);
            return;
        } else if (currentDate.subtract(7, "days").isBefore(myDate)) {
            if (!sortedData.hasOwnProperty("Previous 7 Days")) {
                sortedData["Previous 7 Days"] = [];
            }
            sortedData["Previous 7 Days"].push(data);
            return;
        } else if (currentDate.subtract(30, "days").isBefore(myDate)) {
            if (!sortedData.hasOwnProperty("Previous 30 Days")) {
                sortedData["Previous 30 Days"] = [];
            }
            sortedData["Previous 30 Days"].push(data);
            return;
        } else {
            if (!sortedData.hasOwnProperty(months[month])) {
                sortedData[months[month]] = [];
            }
            sortedData[months[month]].push(data);
        }
    });
    // const history = Object.keys(sortedData);
    return sortedData;
};

export const selectUser = (state: AuthType) => state.user;
export const chatsLength = (state: ChatType) => state.chats.length > 0;
export const isDarkTheme = (state: ThemeType) => state.theme === "dark";
export const isChatSelected = (id: string) => (state: ChatType) =>
    state.chats[0]?.id === id;

export default useChat;
export {useAuth, useSettings, useTheme};

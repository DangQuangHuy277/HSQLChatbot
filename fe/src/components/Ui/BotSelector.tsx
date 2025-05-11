import {IonIcon} from "@ionic/react";
import {sparkles, school, documents, mail, chatbubbleOutline} from "ionicons/icons";
import {AgentList, useSettings} from "../../store/store";
import classNames from "classnames";
import React from "react";

// Define bot types
type BotType = {
    id: AgentList;
    name: string;
    description: string;
    icon: any;
};

export default function BotSelector() {
    const [selectedAgent, setAgent] = useSettings((state) => [
        state.settings.selectedAgent,
        state.setAgent,
    ]);

    // Define available bots
    const availableBots: BotType[] = [
        {
            id: "general",
            name: "General Assistant",
            description: "General purpose chat bot",
            icon: chatbubbleOutline
        },
        {
            id: "study-materials",
            name: "Study Resources",
            description: "Find relevant study materials and documents",
            icon: documents
        },
        {
            id: "academic-advisor",
            name: "Academic Advisor",
            description: "Get course recommendations and academic advice",
            icon: school
        },
        {
            id: "notifications",
            name: "Notifications",
            description: "Send notifications and emails to students",
            icon: mail
        }
    ];

    return (
        <>
            <div
                className="mb-6 md:w-3/4 md:max-w-[600px] mx-2 relative flex flex-col items-center rounded-md mt-5 md:mx-auto">
                <h2 className="text-xl font-bold mb-4 text-gray-800 dark:text-gray-200">
                    Select an Assistant
                </h2>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-3 w-full">
                    {availableBots.map((bot) => (
                        <button
                            key={bot.id}
                            title={bot.name}
                            className={classNames(
                                "bot-selector p-3 transition rounded-lg flex items-center",
                                {
                                    "bg-white dark:bg-dark-primary border-2 dark:border-white/80 border-gray-700":
                                        selectedAgent === bot.id,
                                    "bg-gray-200 dark:bg-[#202123] hover:bg-gray-300 dark:hover:bg-[#2c2d31]":
                                        selectedAgent !== bot.id,
                                }
                            )}
                            onClick={() => setAgent(bot.id)}
                        >
              <span
                  className={classNames("text-2xl mr-3", {
                      "text-teal-500": selectedAgent === bot.id,
                      "text-gray-500 dark:text-gray-400": selectedAgent !== bot.id,
                  })}
              >
                <IonIcon icon={bot.icon}/>
              </span>
                            <div className="text-left">
                                <div className={classNames("font-medium", {
                                    "text-gray-900 dark:text-white": selectedAgent === bot.id,
                                    "text-gray-700 dark:text-gray-300": selectedAgent !== bot.id,
                                })}>
                                    {bot.name}
                                </div>
                                <div className="text-xs text-gray-500 dark:text-gray-400">
                                    {bot.description}
                                </div>
                            </div>
                        </button>
                    ))}
                </div>
            </div>

            <div className="h-60 flex items-start justify-center">
                <h1 className="text-4xl font-bold mt-5 text-center text-gray-800 dark:text-gray-300">
                    University AI Assistant
                </h1>
            </div>
        </>
    );
}
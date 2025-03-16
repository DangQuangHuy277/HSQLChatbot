import {IonIcon} from "@ionic/react";
import {shareOutline, informationCircleOutline} from "ionicons/icons";
import {AgentList, useSettings} from "../../store/store";

export default function Header() {
    const [model, systemMessage, useSystemMessageForAllChats, selectedAgent] = useSettings(
        (state) => [
            state.settings.selectedModal,
            state.settings.systemMessage,
            state.settings.useSystemMessageForAllChats,
            state.settings.selectedAgent,
        ]
    );

    // Map agent IDs to display names
    const agentNames: Record<AgentList, string> =
        {
            "general":
                "General Assistant",
            "study-materials":
                "Study Resources",
            "academic-advisor":
                "Academic Advisor",
            "notifications":
                "Notifications"
        }
    ;

    return (
        <header
            className="text-center my-2 text-sm dark:text-gray-300 border-b dark:border-none dark:shadow-md py-2 flex items-center justify-between px-2">
            {/* Left section - empty for balance */}
            <div className="flex-1">
                {/* Empty to balance the layout */}
            </div>

            {/* Center section - agent name */}
            <div className="flex items-center justify-center">
                    <span
                        className="bg-teal-500/20 text-teal-600 dark:text-teal-400 px-3 py-1 rounded-full font-medium">
                      {agentNames[selectedAgent] || "Assistant"}
                    </span>
            </div>

            {/* Right section - model info and share button */}
            <div className="flex items-center justify-end gap-4 flex-1">
                <div className="flex items-center text-xs text-gray-500 dark:text-gray-400">
                    <span>({model.toLocaleUpperCase()})</span>
                    {useSystemMessageForAllChats && (
                        <span className="flex ml-1 group cursor-pointer">
                            <IonIcon icon={informationCircleOutline} className="text-base"/>
                        <span
                            className="absolute z-10 right-0 w-64 top-[calc(100%+1rem)] text-sm bg-gray-900 text-white p-2 rounded-md invisible pointer-events-none group-hover:visible group-hover:pointer-events-auto transition">
                          <span className="block underline text-teal-600">
                            <strong>System message</strong>
                          </span>
                          <span className="text-gray-400 block text-left">
                            {systemMessage}
                          </span>
                        </span>
                      </span>
                    )}
                </div>
                <button className="text-xl" aria-label="Share">
                    <IonIcon icon={shareOutline}/>
                </button>
            </div>
        </header>
    );
}
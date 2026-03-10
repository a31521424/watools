import {create} from "zustand";
import {createJSONStorage, persist} from "zustand/middleware";
import {RankingInputContext, RankingSelectionRecord, RankingSourceType} from "@/lib/command-ranking";

const MAX_SELECTION_HISTORY = 30;

type RecordSelectionPayload = {
    triggerId: string;
    source: RankingSourceType;
    input: RankingInputContext;
}

type CommandRankingState = {
    history: RankingSelectionRecord[];
    recordSelection: (payload: RecordSelectionPayload) => void;
}

export const useCommandRankingStore = create<CommandRankingState>()(
    persist(
        (set) => ({
            history: [],
            recordSelection: ({triggerId, source, input}) => set(state => ({
                history: [{
                    triggerId,
                    source,
                    inputKey: input.key,
                    normalizedValue: input.normalizedValue,
                    valueType: input.valueType,
                    clipboardContentType: input.clipboardContentType,
                    selectedAt: new Date().toISOString(),
                }, ...state.history].slice(0, MAX_SELECTION_HISTORY)
            })),
        }),
        {
            name: "watools-command-ranking",
            storage: createJSONStorage(() => localStorage),
            partialize: state => ({
                history: state.history
            }),
        }
    )
);

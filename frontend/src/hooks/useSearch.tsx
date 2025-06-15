import {CommandGroupType, mockCommandGroups} from "@/schemas/command";

const useSearch = (keyword?: string): CommandGroupType[] => {
    if (!keyword) {
        return []
    }

    return mockCommandGroups
}

export default useSearch
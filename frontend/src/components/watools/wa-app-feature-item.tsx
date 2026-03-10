import {useMemo} from "react";
import {BaseItemProps} from "@/components/watools/wa-base-item";
import Fuse from "fuse.js";
import {WaIcon} from "@/components/watools/wa-icon";
import {useLocation} from "wouter";
import {compareRankableItems, RankingInputContext, RankingSelectionRecord} from "@/lib/command-ranking";

type UseAppFeatureItemsParams = {
    searchKey: string;
    rankingContext: RankingInputContext;
    rankingHistory: RankingSelectionRecord[];
    onTriggerAppFeature: () => void;
}

// 应用内置功能定义（纯前端快捷操作）
const LOCAL_APP_FEATURES = [
    {
        id: "app-feature-plugin-management",
        name: "Plugin Management",
        description: "Manage installed plugins",
        icon: "puzzle",
        navigatePath: "/plugin-management",
        keywords: ["plugin", "extension", "manage", "installation"]
    },
];

export const useAppFeatureItems = ({
    searchKey,
    rankingContext,
    rankingHistory,
    onTriggerAppFeature
}: UseAppFeatureItemsParams) => {
    const [_, navigate] = useLocation();

    const appFeatureFuse = useMemo(() => {
        return new Fuse(LOCAL_APP_FEATURES, {
            threshold: 0.4,
            minMatchCharLength: 1,
            useExtendedSearch: true,
            ignoreLocation: true,
            shouldSort: false,
            keys: [
                {name: 'name', weight: 1.0},
                {name: 'keywords', weight: 0.8}
            ]
        });
    }, []);

    return useMemo((): BaseItemProps[] => {
        if (!searchKey) {
            return [];
        }

        const results = appFeatureFuse.search(searchKey, {limit: 5})
            .map((result, index) => ({
                feature: result.item,
                rankingMeta: {
                    source: "app-feature" as const,
                    sourceOrder: index,
                }
            }))
            .sort((a, b) => compareRankableItems({
                triggerId: a.feature.id,
                title: a.feature.name,
                rankingMeta: a.rankingMeta,
            }, {
                triggerId: b.feature.id,
                title: b.feature.name,
                rankingMeta: b.rankingMeta,
            }, rankingContext, rankingHistory));

        return results.map(({feature, rankingMeta}) => {
            return {
                id: feature.id,
                triggerId: feature.id,
                title: feature.name,
                icon: <WaIcon value={feature.icon} size={16}/>,
                usedCount: 0,
                rankingMeta,
                subtitle: feature.description,
                badge: "App",
                onSelect: () => {
                    navigate(feature.navigatePath);
                    onTriggerAppFeature()
                }
            };
        });
    }, [searchKey, appFeatureFuse, navigate, onTriggerAppFeature, rankingContext, rankingHistory]);

};

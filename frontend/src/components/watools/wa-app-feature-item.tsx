import {useMemo} from "react";
import {BaseItemProps} from "@/components/watools/wa-base-item";
import Fuse from "fuse.js";
import {WaIcon} from "@/components/watools/wa-icon";
import {useLocation} from "wouter";

type UseAppFeatureItemsParams = {
    searchKey: string;
}

// 应用内置功能定义（纯前端快捷操作）
const LOCAL_APP_FEATURES = [
    {
        id: "app-feature-plugin-management",
        name: "Plugin Management",
        description: "Manage installed plugins",
        icon: "puzzle",
        navigatePath: "/plugin-management",
        keywords: ["Plugins"]
    },
    // 未来可以添加更多应用功能，例如：
    // - Settings / Preferences（设置/偏好）
    // - About（关于）
    // - Help / Documentation（帮助/文档）
    // - Update（更新）
];

export const useAppFeatureItems = ({searchKey}: UseAppFeatureItemsParams) => {
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

    const filteredItems = useMemo((): BaseItemProps[] => {
        if (!searchKey) {
            return [];
        }

        const results = appFeatureFuse.search(searchKey, {limit: 5});
        return results.map(result => {
            const feature = result.item;
            return {
                id: feature.id,
                triggerId: feature.id,
                title: feature.name,
                icon: <WaIcon value={feature.icon} size={16}/>,
                usedCount: 0,
                subtitle: feature.description,
                badge: "App",
                onSelect: () => {
                    navigate(feature.navigatePath);
                }
            };
        });
    }, [searchKey, appFeatureFuse, navigate]);

    return filteredItems;
};

import {useEffect, useState} from "react";
import {CommandGroupType, CommandType, OperationCommandType} from "@/schemas/command";
import {WaBaseCommandGroup} from "@/components/watools/wa-base-command-group";
import {IFuseOptions} from "fuse.js";
import {WaIcon} from "@/components/watools/wa-icon";
import {getOperationCommands} from "@/api/command";

type WaOperationCommandGroupProps = {
    searchKey: string
    onTriggerCommand: (command: CommandType) => void
    onSearchSuccess: (selectedKey?: string) => void
}

const WaBaseCommandFuseConfig: IFuseOptions<OperationCommandType> = {
    threshold: 0.3,
    minMatchCharLength: 1,
    useExtendedSearch: true,
    ignoreLocation: true,
    keys: [{
        name: 'name',
        weight: 1.0
    }]
}


export const WaOperationCommandGroup = (props: WaOperationCommandGroupProps) => {
    const [operationCommandGroup, setOperationCommandGroup] = useState<CommandGroupType<OperationCommandType> | null>(null)
    const initApplication = () => {
        getOperationCommands().then(setOperationCommandGroup)
    }
    useEffect(() => {
        initApplication()
    }, [])
    if (!props.searchKey) {
        return null
    }
    if (!operationCommandGroup) {
        return null
    }

    return <WaBaseCommandGroup<OperationCommandType>
        searchKey={props.searchKey}
        commandGroup={operationCommandGroup}
        onTriggerCommand={props.onTriggerCommand}
        onSearchSuccess={props.onSearchSuccess}
        fuseOptions={WaBaseCommandFuseConfig}
        renderItemIcon={command => <WaIcon value={command.icon} size={16}/>}
    />
}
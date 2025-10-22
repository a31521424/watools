import {useEffect, useState} from "react";
import {ApplicationCommandType, CommandGroupType, CommandType} from "@/schemas/command";
import {WaBaseCommandGroup} from "@/components/watools/wa-base-command-group";
import {IFuseOptions} from "fuse.js";
import {getApplicationCommands} from "@/api/command";
import {WaIcon} from "@/components/watools/wa-icon";
import {EventsOff, EventsOn} from "../../../wailsjs/runtime";

type WaApplicationCommandGroupProps = {
    searchKey: string
    onTriggerCommand: (command: CommandType) => void
    onSearchSuccess: (selectedKey?: string) => void
}

const WaBaseCommandFuseConfig: IFuseOptions<ApplicationCommandType> = {
    threshold: 0.3,
    minMatchCharLength: 1,
    useExtendedSearch: true,
    ignoreLocation: true,
    keys: [{
        name: 'name',
        weight: 1.0
    }, {
        name: 'nameInitial',
        weight: 0.8
    }, {
        name: 'pathName',
        weight: 0.6
    }]
}


export const WaApplicationCommandGroup = (props: WaApplicationCommandGroupProps) => {
    const [applicationCommandGroup, setApplicationCommandGroup] = useState<CommandGroupType<ApplicationCommandType> | null>(null)
    useEffect(() => {
        getApplicationCommands().then(setApplicationCommandGroup)
        EventsOn('watools.applicationChanged', () => {
            getApplicationCommands().then(setApplicationCommandGroup)
        })
        return () => {
            EventsOff('watools.applicationChanged')
        }
    }, [])
    if (!props.searchKey) {
        return null
    }
    if (!applicationCommandGroup) {
        return null
    }

    return <WaBaseCommandGroup<ApplicationCommandType>
        searchKey={props.searchKey}
        commandGroup={applicationCommandGroup}
        onTriggerCommand={props.onTriggerCommand}
        onSearchSuccess={props.onSearchSuccess}
        fuseOptions={WaBaseCommandFuseConfig}
        renderItemIcon={command => <WaIcon
            iconPath={`/api/application-icon?path=${encodeURIComponent(command.iconPath)}`} size={16}/>
        }
    />
}
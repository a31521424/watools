import {useEffect, useState} from "react";
import {ApplicationCommandType, CommandGroupType, CommandType} from "@/schemas/command";
import {WaBaseCommandGroup} from "@/components/watools/wa-base-command-group";
import {GetApplications} from "../../../wailsjs/go/launch/WaLaunchApp";
import {isContainNonAscii, toPinyinInitial} from "@/lib/search";
import {IFuseOptions} from "fuse.js";

type WaApplicationCommandGroupProps = {
    searchKey: string
    onTriggerCommand: (command: CommandType) => void
    onSearchSuccess: () => void
}

const WaBaseCommandFuseConfig: IFuseOptions<ApplicationCommandType> = {
    threshold: 0.3,
    minMatchCharLength: 1,
    // includeScore: true,
    // includeMatches: true,
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
    const initApplication = () => {
        GetApplications().then(res => {
            console.log('fetch application', res)
            if (res == null) {
                return
            }
            setApplicationCommandGroup({
                category: 'Application',
                commands: res.map(command => ({
                    ...command,
                    category: 'Application',
                    nameInitial: isContainNonAscii(command.name) ? toPinyinInitial(command.name) : null,
                    pathName: command.path.split('/').pop() || ''
                }))
            })
        })
    }
    useEffect(() => {
        initApplication()
    }, [])
    if (!props.searchKey) {
        return null
    }
    if (!applicationCommandGroup) {
        return null
    }

    return <WaBaseCommandGroup
        searchKey={props.searchKey}
        commandGroup={applicationCommandGroup}
        onTriggerCommand={props.onTriggerCommand}
        onSearchSuccess={props.onSearchSuccess}
        fuseOptions={WaBaseCommandFuseConfig}
    />
}
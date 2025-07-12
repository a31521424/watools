import {useEffect, useState} from "react";
import {CommandGroupType, CommandType} from "@/schemas/command";
import {WaBaseCommandGroup} from "@/components/watools/wa-base-command-group";
import {GetApplication} from "../../../wailsjs/go/launch/WaLaunchApp";

type WaApplicationCommandGroupProps = {
    searchKey: string
    onTriggerCommand: (command: CommandType) => void
}


export const WaApplicationCommandGroup = (props: WaApplicationCommandGroupProps) => {
    const [applicationCommandGroup, setApplicationCommandGroup] = useState<CommandGroupType | null>(null)
    const initApplication = () => {
        GetApplication().then(res => {
            console.log('fetch application', res)
            if (res == null) {
                return
            }
            setApplicationCommandGroup({
                category: 'Application',
                // @ts-ignore
                commands: res
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
    />
}
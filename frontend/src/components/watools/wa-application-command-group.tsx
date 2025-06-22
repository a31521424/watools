import {useEffect, useState} from "react";
import {CommandGroupType, CommandType} from "@/schemas/command";
import {WaBaseCommandGroup} from "@/components/watools/wa-base-command-group";
import {GetApplication, RunApplication} from "../../../wailsjs/go/launch/WaLaunchApp";

type WaApplicationCommandGroupProps = {
    searchKey: string
}


export const WaApplicationCommandGroup = (props: WaApplicationCommandGroupProps) => {
    const [applicationCommandGroup, setApplicationCommandGroup] = useState<CommandGroupType | null>(null)
    const initApplication = () => {
        GetApplication().then(res => {
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
    const onTriggerCommand = (command: CommandType) => {
        RunApplication(command.path).then(res => {
            console.log(res)
        })
    }
    return <WaBaseCommandGroup
        searchKey={props.searchKey}
        commandGroup={applicationCommandGroup}
        onTriggerCommand={onTriggerCommand}
    />
}
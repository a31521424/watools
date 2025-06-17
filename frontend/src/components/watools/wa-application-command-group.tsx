import {useEffect, useState} from "react";
import {CommandGroupType} from "@/schemas/command";
import {GetSystemApplication} from "../../../wailsjs/go/apps/WaApp";
import {WaBaseCommandGroup} from "@/components/watools/wa-base-command-group";

type WaApplicationCommandGroupProps = {
    searchKey: string
}


export const WaApplicationCommandGroup = (props: WaApplicationCommandGroupProps) => {
    const [applicationCommandGroup, setApplicationCommandGroup] = useState<CommandGroupType | null>(null)
    const initApplication = () => {
        GetSystemApplication().then(res => {
            // @ts-ignore
            setApplicationCommandGroup(res)
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
    return <WaBaseCommandGroup searchKey={props.searchKey} commandGroup={applicationCommandGroup}/>
}
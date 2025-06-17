import {ReactNode, useEffect, useState} from "react";
import {DynamicIcon, IconName} from "lucide-react/dynamic";
import {GetIconBase64} from "../../../wailsjs/go/apps/WaApp";

type WaIconProps = {
    value?: IconName | ReactNode | null
    color?: string
    size?: number | string
    iconPath?: string

}


export const WaIcon = (props: WaIconProps) => {
    const [iconData, setIconData] = useState<string | null>(null)
    useEffect(() => {
        if (props.iconPath) {
            GetIconBase64(props.iconPath).then(res => {
                setIconData(`data:image/png;base64,${res}`)
            })
        }
    }, [props.iconPath])

    if (typeof props.value === 'string') {
        return <DynamicIcon name={props.value as IconName} color={props.color} size={props.size}/>
    }
    if (props.value) {
        return <>{props.value}</>
    }
    if (props.iconPath && iconData) {
        return <img src={iconData} alt=""/>
    }
    return <></>

}
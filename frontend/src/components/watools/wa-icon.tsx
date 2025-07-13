import {ReactNode} from "react";
import {DynamicIcon, IconName} from "lucide-react/dynamic";

type WaIconProps = {
    value?: IconName | ReactNode | null
    color?: string
    size?: number | string
    iconPath?: string

}


export const WaIcon = (props: WaIconProps) => {
    let iconUrl = ''
    if (props.iconPath) {
        iconUrl = `/api/application-icon?path=${encodeURIComponent(props.iconPath)}`
    }

    if (typeof props.value === 'string') {
        return <DynamicIcon name={props.value as IconName} color={props.color} size={props.size}/>
    }
    if (props.value) {
        return <>{props.value}</>
    }
    if (iconUrl) {
        return <img className="w-6 h-6" src={iconUrl} alt=""/>
    }
    return <></>

}
import {ReactNode} from "react";
import {DynamicIcon, IconName} from "lucide-react/dynamic";

type WaIconProps = {
    value?: IconName | ReactNode | null
    color?: string
    size?: number | string
    iconPath?: string

}


export const WaIcon = (props: WaIconProps) => {
    if (props.iconPath) {
        return <img className="w-6 h-6" src={props.iconPath} alt="Plugins Icon"/>
    }
    if (typeof props.value === 'string') {
        return <DynamicIcon name={props.value as IconName} color={props.color} size={props.size}/>
    }
    if (props.value) {
        return <>{props.value}</>
    }
    return <></>

}
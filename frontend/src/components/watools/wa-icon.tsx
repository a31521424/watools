import {ReactNode} from "react";
import {DynamicIcon, IconName} from "lucide-react/dynamic";

type WaIconProps = {
    value?: IconName | ReactNode | null
    color?: string
    size?: number | string

}

const isNotPureAscii = (char: string) => {
    return char.split('').some(c => !(c.charCodeAt(0) < 128))
}

export const WaIcon = (props: WaIconProps) => {
    if (!props.value) {
        return <></>
    }
    if (typeof props.value === 'string') {
        if (isNotPureAscii(props.value)) {
            return <div>{props.value}</div>
        }
        return <DynamicIcon name={props.value as IconName} color={props.color} size={props.size}/>
    }
    return <>{props.value}</>
}
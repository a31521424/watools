import {Command, CommandEmpty, CommandGroup, CommandItem, CommandList} from "../ui/command";
import {WaComplexInput} from "./wa-complex-input";
import useResizeWindow from "@/hooks/useResizeWindow";
import {useEffect, useState} from "react";
import {CommandGroupType} from "@/schemas/command";
import {WaIcon} from "@/components/watools/wa-icon";
import {GetSystemApplication} from "../../../wailsjs/go/apps/WaApp";

const Main = () => {
    const windowRef = useResizeWindow<HTMLDivElement>()
    const [input, setInput] = useState<string>('')
    const [searchResult, setSearchResult] = useState<CommandGroupType[]>([])

    const initApplication = () => {
        GetSystemApplication().then(res => {
            // @ts-ignore
            setSearchResult([res])
        })
    }

    useEffect(() => {
        initApplication()
    }, [])


    return <div ref={windowRef} className="bg-white w-full rounded-xl overflow-x-hidden scrollbar-hide">
        <Command className="rounded-lg border shadow-md md:min-w-[450px] p-2">
            <WaComplexInput
                onValueChange={setInput}
                classNames={{wrapper: !!input ? undefined : "!border-none"}}
            />
            <CommandList className={!!input ? undefined : "hidden"}>
                <CommandEmpty>No results found.</CommandEmpty>
                {searchResult.map(group => (
                    <CommandGroup key={group.category} heading={group.category}>
                        {group.commands.map(command => (
                            <CommandItem
                                key={command.name}
                                value={`${command.category.toLowerCase()}-${command.name.toLowerCase()}`}
                                className='gap-x-4'
                            >
                                <WaIcon value={command.icon} iconPath={command.iconPath}/>
                                <span>{command.name}</span>
                            </CommandItem>
                        ))}
                    </CommandGroup>
                ))}
            </CommandList>
        </Command>
    </div>
}


export default Main
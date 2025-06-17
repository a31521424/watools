import {WaComplexInput} from "@/components/watools/wa-complex-input";
import {Command, CommandEmpty, CommandList} from "@/components/ui/command";
import {useState} from "react";
import {WaApplicationCommandGroup} from "@/components/watools/wa-application-command-group";


export const WaCommand = () => {
    const [input, setInput] = useState<string>('')

    return <Command
        shouldFilter={false}
        className="rounded-lg border shadow-md w-full p-2"
    >
        <WaComplexInput
            onValueChange={setInput}
            classNames={{wrapper: !!input ? undefined : "!border-none"}}
        />
        <CommandList className={!!input ? undefined : "hidden"}>
            <CommandEmpty>No results found.</CommandEmpty>
            <WaApplicationCommandGroup searchKey={input}/>
        </CommandList>
    </Command>
}
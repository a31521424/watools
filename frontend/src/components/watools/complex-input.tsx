import {Input} from "@/components/ui/input";
import {cn} from "@/lib/utils";

const ComplexInput = () => {
    return <div className="flex flex-row text-5xl items-center gap-x-2">

        <Input
            className={cn(
                "bg-transparent border-transparent shadow-none",
                "focus-visible:ring-0 focus-visible:ring-offset-0 focus-visible:border-transparent",
                "md:text-1xl text-2xl",
                "h-auto"
            )}
            placeholder="Hello Watools"
        />
    </div>
}

export default ComplexInput
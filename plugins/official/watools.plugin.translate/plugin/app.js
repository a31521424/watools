const entry = [
    {
        type: "ui",
        subTitle: "Open Translation Panel",
        icon: "languages",
        match: (context) => {
            const input = context.input.value.trim().toLowerCase();
            if (!input) {
                return false;
            }

            return (
                input === "fy" ||
                input === "translate" ||
                input === "翻译" ||
                input.startsWith("fy ") ||
                input.startsWith("translate ") ||
                input.startsWith("翻译 ")
            );
        },
        file: "index.html"
    }
];

export default entry;

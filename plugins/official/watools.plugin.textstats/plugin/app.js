const triggers = ["textcount", "count", "chars", "字符统计", "字数统计", "文本统计"];

const escapeRegExp = (value) => value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

const triggerPattern = new RegExp(
    `^(?:${triggers.map(escapeRegExp).join("|")})(?:\\s+[\\s\\S]*)?$`,
    "i"
);

const entry = [{
    type: "ui",
    subTitle: "Open Text Statistics Panel",
    icon: "file-text",
    match: (context) => {
        const input = context.input.value.trim();
        if (!input) {
            return false;
        }

        return triggerPattern.test(input);
    },
    file: "index.html"
}];

export default entry;

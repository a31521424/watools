const triggers = ["json", "url2params", "query2params", "qs2params"];

const escapeRegExp = (value) => value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

const triggerPattern = new RegExp(
    `^(?:${triggers.map(escapeRegExp).join("|")})(?:\\s+[\\s\\S]*)?$`,
    "i"
);

const entry = [{
    type: "ui",
    subTitle: "打开 JSON 编辑器",
    icon: "braces",
    match: (context) => {
        const input = (context.input.value || "").trim();
        if (!input) {
            return false;
        }

        return triggerPattern.test(input);
    },
    file: "index.html"
}];

export default entry;

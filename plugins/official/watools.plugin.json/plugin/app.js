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
    matchPreview: (context) => {
        const head = context.input.head.trimStart();
        const tail = context.input.tail.trimEnd();
        return (head.startsWith("{") && tail.endsWith("}"))
            || (head.startsWith("[") && tail.endsWith("]"))
            || triggerPattern.test(head);
    },
    file: "index.html"
}];

export default entry;

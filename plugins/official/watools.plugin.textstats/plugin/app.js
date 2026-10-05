const triggers = ["textcount", "count", "chars", "字符统计", "字数统计", "文本统计"];

const escapeRegExp = (value) => value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

const triggerPattern = new RegExp(
    `^(?:${triggers.map(escapeRegExp).join("|")})(?:\\s+[\\s\\S]*)?$`,
    "i"
);

const entry = [{
    type: "ui",
    subTitle: "打开文本统计面板",
    icon: "file-text",
    matchPreview: (context) => context.input.length > 2000 || triggerPattern.test(context.input.head.trim()),
    file: "index.html"
}];

export default entry;

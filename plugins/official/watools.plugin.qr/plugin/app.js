const triggers = ["qr", "qrcode", "二维码"];

const escapeRegExp = (value) => value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

const triggerPattern = new RegExp(
    `^(?:${triggers.map(escapeRegExp).join("|")})(?:\\s+[\\s\\S]*)?$`,
    "i"
);

const entry = [{
    type: "ui",
    subTitle: "打开二维码工作区",
    icon: "qr-code",
    matchPreview: (context) => context.input.clipboardContentType === "image" || triggerPattern.test(context.input.head.trim()),
    file: "index.html"
}];

export default entry;

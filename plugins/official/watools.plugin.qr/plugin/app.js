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

const triggers = ["qr", "qrcode", "二维码"];

const escapeRegExp = (value) => value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

const triggerPattern = new RegExp(
    `^(?:${triggers.map(escapeRegExp).join("|")})(?:\\s+[\\s\\S]*)?$`,
    "i"
);

const entry = [{
    type: "ui",
    subTitle: "Open QR Workspace",
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

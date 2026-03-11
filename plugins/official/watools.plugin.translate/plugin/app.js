const hasTranslationPrefix = (value) => {
    return /^(fy|translate|翻译)(\s+.*)?$/i.test(value);
};

const hasTranslatableContent = (value) => {
    if (!value || value.length < 2) {
        return false;
    }

    if (/^(https?:\/\/|www\.|\/|~\/)/i.test(value)) {
        return false;
    }

    return /[A-Za-z\u3400-\u9FFF]/.test(value);
};

const entry = [{
    type: "ui",
    subTitle: "打开翻译面板",
    icon: "languages",
    match: (context) => {
        const input = context.input.value.trim();
        if (!input) {
            return false;
        }

        if (hasTranslationPrefix(input)) {
            return true;
        }

        return hasTranslatableContent(input);
    },
    file: "index.html"
}];

export default entry;

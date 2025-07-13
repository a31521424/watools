import {pinyin} from "pinyin-pro";

export const isContainNonAscii = (str: string) => {
    const charIsNonAscii = (char: string) => {
        const charCode = char.charCodeAt(0)
        return 0 > charCode || charCode > 127
    }
    if (str.length === 1) {
        return charIsNonAscii(str)
    }
    return str.split("").some(charIsNonAscii)

}

export const toPinyin = (str: string): string => {
    return pinyin(str, {
        toneType: 'none',
        type: 'string',
    }).replace(/\s/g, '').toLowerCase()
}

export const toPinyinInitial = (str: string): string => {
    return pinyin(str, {
        pattern: 'first',
        toneType: 'none',
        type: 'string',
    }).replace(/\s/g, '').toLowerCase()
}
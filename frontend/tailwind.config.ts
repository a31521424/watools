const {fontFamily} = require("tailwindcss/defaultTheme")

const config = {
    darkMode: ["class"],
    content: [
        './pages/**/*.{ts,tsx}',
        './components/**/*.{ts,tsx}',
        './app/**/*.{ts,tsx}',
        './src/**/*.{ts,tsx}'
    ],
    prefix: "",
    theme: {
        extends: {
            fontFamily: {
                sans: [
                    "Inter",
                    "Noto Sans SC",
                    ...fontFamily.sans
                ]
            }
        }
    },
    plugins: [
        require("tailwindcss-animate"),
        require('tailwind-scrollbar-hide')
    ],
}

export default config
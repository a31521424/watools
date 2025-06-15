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
                    'system-ui',
                    '-apple-system',
                    'BlinkMacSystemFont',
                    '"Segoe UI"',
                    'Roboto',
                    '"Helvetica Neue"',
                    '"PingFang SC"',
                    '"Microsoft YaHei"',
                    'sans-serif',
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
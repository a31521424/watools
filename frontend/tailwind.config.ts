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
        // ... (您的 theme 設定，如果有的話)
    },
    plugins: [
        require("tailwindcss-animate"),
        require('tailwind-scrollbar-hide')
    ],
}

export default config
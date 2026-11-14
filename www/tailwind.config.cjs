const coreConfig = require("./tailwind/modified.tailwind.config.cjs")

/** @type {import('tailwindcss').Config} */
module.exports = {
  ...coreConfig,
  content: [
    "./vendor/**/*.{ts,tsx}",
    "./src/**/*.{astro,ts,tsx,mdx,md}",
    "./content/**/*.{md,mdx}",
  ],
  theme: {
    ...coreConfig.theme,
    extend: {
      ...coreConfig.theme.extend,
      container: { center: true, screens: { "2xl": "1400px" } },
    },
  },
}

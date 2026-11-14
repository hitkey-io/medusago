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
    backgroundImage: {
      ...coreConfig.theme.backgroundImage,
      "search-hit": "url('/images/search-hit-light.svg')",
      "search-hit-dark": "url('/images/search-hit.svg')",
      "search-arrow": "url('/images/search-hit-arrow-light.svg')",
      "search-arrow-dark": "url('/images/search-hit-arrow.svg')",
      "search-no-result": "url('/images/search-no-result-light.svg')",
      "search-no-result-dark": "url('/images/search-no-result.svg')",
      "magnifying-glass": "url('/images/magnifying-glass.svg')",
      "magnifying-glass-dark": "url('/images/magnifying-glass-dark.svg')",
      "bg-stripes": "url('/images/bg-stripes.png')",
      "bg-stripes-dark": "url('/images/bg-stripes-dark.png')",
    },
  },
}

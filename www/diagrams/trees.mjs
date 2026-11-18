// Directory trees (rendered by src/components/FileTree.tsx), the replacement of the "directory structure" pictures of the upstream pages.
// A tree is one entry per line, two spaces per level; " *" marks the entry the page is about. Like the diagrams, `pages[page][n]` is the
// n-th picture of the upstream page (`null` where the picture is a screenshot).
const app = (tree) => ({ title: "MedusaGo Application", tree })
const next = (tree) => ({ title: "Next.js Storefront", tree })

const sdk = app("src\n  admin\n    lib\n      sdk.ts *")

export const pages = {
  "learn/customization/customize-admin/widget": [
    sdk,
    app("src\n  admin\n    lib\n      sdk.ts\n    widgets\n      product-brand.tsx *"),
    null,
  ],
  "learn/customization/customize-admin/route": [
    sdk,
    app("src\n  admin\n    lib\n      sdk.ts\n    routes\n      brands\n        page.tsx *"),
    null,
  ],
  "learn/fundamentals/admin/translations": [
    app("src\n  admin\n    i18n\n      json\n        en.json *"),
    app("src\n  admin\n    i18n\n      json\n        en.json\n        es.json *"),
    app("src\n  admin\n    i18n\n      index.ts *\n      json\n        en.json\n        es.json"),
  ],
  "learn/fundamentals/admin/ui-routes": [
    app("src\n  admin\n    routes\n      custom\n        page.tsx *"),
    app("src\n  admin\n    routes\n      custom\n        page.tsx\n        nested\n          page.tsx *"),
    app("src\n  admin\n    routes\n      settings\n        custom\n          page.tsx *"),
    app("src\n  admin\n    routes\n      custom\n        [id]\n          page.tsx *"),
  ],
  "learn/fundamentals/admin/widgets": [app("src\n  admin\n    widgets\n      product-widget.tsx *")],
  "resources/nextjs-starter/guides/remove-country-code": [
    {
      trees: [
        { title: "Before", tree: "src\n  app\n    [countryCode]\n      (checkout)\n      (main)" },
        { title: "After", tree: "src\n  app\n    (checkout)\n    (main)" },
      ],
    },
  ],
  // pictures: the result, lib/sdk.ts, the publishable key (admin screenshot), then one tree per created file, with screenshots between
  "resources/storefront-development/guides/express-checkout": [
    null,
    next("lib\n  sdk.ts *"),
    null,
    next("providers\n  region.tsx *"),
    next("providers\n  cart.tsx *\n  region.tsx"),
    next("components\n  SecondCol\n    index.tsx *"),
    next("components\n  Router\n    index.tsx *\n  SecondCol\n    index.tsx"),
    next("app\n  [handle]\n    page.tsx *"),
    next("components\n  Card\n    index.tsx *\n  Router\n    index.tsx\n  SecondCol\n    index.tsx"),
    next("components\n  Card\n    index.tsx\n  Product\n    index.tsx *\n  Router\n    index.tsx\n  SecondCol\n    index.tsx"),
    next("lib\n  price.ts *\n  sdk.ts"),
    null,
    next("components\n  Address\n    index.tsx *\n  Card\n    index.tsx\n  Product\n    index.tsx\n  Router\n    index.tsx\n  SecondCol\n    index.tsx"),
    null,
    next("components\n  Address\n    index.tsx\n  Card\n    index.tsx\n  Product\n    index.tsx\n  Router\n    index.tsx\n  SecondCol\n    index.tsx\n  Shipping\n    index.tsx *"),
    null,
    next("components\n  Address\n    index.tsx\n  Card\n    index.tsx\n  Payment\n    index.tsx *\n  Product\n    index.tsx\n  Router\n    index.tsx\n  SecondCol\n    index.tsx\n  Shipping\n    index.tsx"),
    null,
    next("app\n  confirmation\n    [id]\n      page.tsx *"),
    null,
  ],
}

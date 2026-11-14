// One entry per picture. `url` is the upstream picture the shot replaces (the importer puts our
// file at the place where the upstream page had it); `id` is the file name in public/images/shots.
// path: dashboard route; wait: ms after load; run(page): extra steps (open a menu, fill a field...);
// clip: crop in CSS px (default: the whole 1435x807 window).
const CDN = "https://res.cloudinary.com/dza7lstvk/image/upload/"
export const shots = [
  { id: "orders-list", path: "/orders", url: CDN + "v1739542797/User%20Guide/Screenshot_2025-02-14_at_4.19.37_PM_czal8i.png" },
  { id: "products-list", path: "/products", url: CDN + "v1739530936/User%20Guide/Screenshot_2025-02-14_at_1.01.35_PM_ecyltl.png" },
  { id: "settings-store", path: "/settings/store", url: CDN + "v1739962494/User%20Guide/Screenshot_2025-02-19_at_12.54.00_PM_eozum8.png" },
]

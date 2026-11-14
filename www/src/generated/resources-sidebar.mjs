export const generatedSidebars = [
  {
    "sidebar_id": "recipes",
    "title": "Recipes",
    "items": []
  },
  {
    "sidebar_id": "integrations",
    "title": "Integrations",
    "items": [
      {
        "loaded": true,
        "type": "category",
        "title": "Auth",
        "initialOpen": true,
        "children": [
          {
            "loaded": true,
            "type": "ref",
            "path": "/resources/commerce-modules/auth/auth-providers/google",
            "title": "Google",
            "children": []
          },
          {
            "loaded": true,
            "type": "ref",
            "path": "/resources/commerce-modules/auth/auth-providers/github",
            "title": "GitHub",
            "children": []
          }
        ]
      },
      {
        "loaded": true,
        "type": "category",
        "title": "Payment",
        "initialOpen": true,
        "children": [
          {
            "loaded": true,
            "type": "ref",
            "path": "/resources/commerce-modules/payment/payment-provider/stripe",
            "title": "Stripe",
            "children": []
          },
          {
            "loaded": true,
            "type": "ref",
            "path": "/resources/commerce-modules/payment/payment-provider/adyen",
            "title": "Adyen",
            "children": []
          }
        ]
      }
    ]
  },
  {
    "sidebar_id": "storefront-development",
    "title": "Storefront Development",
    "items": [
      {
        "loaded": true,
        "type": "link",
        "path": "/resources/storefront-development",
        "title": "Overview",
        "children": []
      },
      {
        "type": "separator"
      },
      {
        "loaded": true,
        "type": "category",
        "title": "General",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/tips",
            "title": "Connect to Medusa",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/publishable-api-keys",
            "title": "Publishable API Key",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/localization",
            "title": "Localization",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/production-optimizations",
            "title": "Production Optimizations",
            "children": []
          }
        ]
      },
      {
        "loaded": true,
        "type": "category",
        "title": "Tutorials",
        "description": "The following tutorials guide you to build a storefront for different kinds of use cases.",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/guides/express-checkout",
            "title": "Express Checkout Storefront",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/guides/react-native-expo",
            "title": "Mobile App with React Native and Expo",
            "children": []
          }
        ]
      },
      {
        "loaded": true,
        "type": "category",
        "title": "Regions",
        "description": "Regions are different areas you sell products in. Learn how to use regions in your storefront.",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/regions",
            "title": "Overview",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/regions/list",
            "title": "List Regions",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/regions/store-retrieve-region",
            "title": "Store and Retrieve Regions",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/regions/context",
            "title": "Region React Context",
            "children": []
          }
        ]
      },
      {
        "loaded": true,
        "type": "category",
        "title": "Products",
        "description": "Learn how to show products and related data in your storefront.",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/products/list",
            "title": "List Products",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/storefront-development/products/options",
                "title": "Filter by Options",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/products/retrieve",
            "title": "Retrieve a Product",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/products/variants",
            "title": "Select a Variant",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/products/price",
            "title": "Retrieve Variant Prices",
            "autogenerate_path": "storefront-development/products/price/examples",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/storefront-development/products/price/examples/show-price",
                "title": "Example: Show Variant's Price",
                "description": "",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/storefront-development/products/price/examples/sale-price",
                "title": "Example: Show Sale Price",
                "description": "",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/storefront-development/products/price/examples/tax-price",
                "title": "Example: Show Price with Taxes",
                "description": "",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/products/inventory",
            "title": "Retrieve Variant Inventory",
            "children": []
          }
        ]
      },
      {
        "loaded": true,
        "type": "category",
        "title": "Product Categories",
        "description": "Learn how to show product categories in your storefront.",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/products/categories/list",
            "title": "List Categories",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/products/categories/retrieve",
            "title": "Retrieve a Category",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/products/categories/products",
            "title": "Retrieve a Category's Products",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/products/categories/nested-categories",
            "title": "Retrieve Nested Categories",
            "children": []
          }
        ]
      },
      {
        "loaded": true,
        "type": "category",
        "title": "Product Collections",
        "description": "Learn how to show product collections in your storefront.",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/products/collections/list",
            "title": "List Collections",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/products/collections/retrieve",
            "title": "Retrieve a Collection",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/products/collections/products",
            "title": "Retrieve a Collection's Products",
            "children": []
          }
        ]
      },
      {
        "loaded": true,
        "type": "category",
        "title": "Carts",
        "description": "Learn how to manage carts in your storefront. These carts are later used to implement the checkout flow.",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/cart/create",
            "title": "Create Cart",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/cart/retrieve",
            "title": "Retrieve Cart",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/cart/context",
            "title": "Cart React Context",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/cart/update",
            "title": "Update Cart",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/cart/manage-items",
            "title": "Manage Line Items",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/cart/manage-promotions",
            "title": "Manage Promotions",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/cart/totals",
            "title": "Show Totals",
            "children": []
          }
        ]
      },
      {
        "loaded": true,
        "type": "category",
        "title": "Checkout",
        "description": "Learn how to implement the different steps of the checkout flow in your storefront. By following the checkout flow, customers can complete their purchase.",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/checkout",
            "title": "Overview",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/checkout/email",
            "title": "1. Enter Email",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/checkout/address",
            "title": "2. Set Address",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/checkout/shipping",
            "title": "3. Choose Shipping Method",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/checkout/payment",
            "title": "4. Choose Payment Provider",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/storefront-development/checkout/payment/stripe",
                "title": "Example: Stripe",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/checkout/complete-cart",
            "title": "5. Complete Cart",
            "children": []
          },
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/storefront-development/checkout/order-confirmation",
            "title": "Show Order Confirmation",
            "children": []
          }
        ]
      },
      {
        "loaded": true,
        "type": "category",
        "title": "Customers",
        "description": "Learn how to allow customers to register, login, and manage their profile in your storefront.",
        "children": [
          {
            "loaded": true,
            "type": "sub-category",
            "title": "Authentication",
            "description": "Learn how to authenticate customers in your storefront. This includes basic email-password authentication and third-party (social) login.",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/storefront-development/customers/register",
                "title": "Register Customer",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/storefront-development/customers/verify-account",
                "title": "Register Customer with Email Verification",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/storefront-development/customers/login",
                "title": "Login Customer",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/storefront-development/customers/third-party-login",
                "title": "Third-Party (Social) Login",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/storefront-development/customers/reset-password",
                "title": "Reset Password",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "sub-category",
            "title": "Profile",
            "description": "Learn how to retrieve a customer, manage their customer profile, and log them out in your storefront.",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/storefront-development/customers/retrieve",
                "title": "Retrieve Customer",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/storefront-development/customers/context",
                "title": "Customer React Context",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/storefront-development/customers/profile",
                "title": "Edit Customer Profile",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/storefront-development/customers/addresses",
                "title": "Manage Customer Addresses",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/storefront-development/customers/log-out",
                "title": "Log-out Customer",
                "children": []
              }
            ]
          }
        ]
      }
    ]
  },
  {
    "sidebar_id": "tools",
    "title": "Tools",
    "items": [
      {
        "loaded": true,
        "type": "category",
        "title": "CLI Tools",
        "initialOpen": true,
        "description": "CLI tools help you setup Medusa, manage the database, and more.",
        "children": [
          {
            "loaded": true,
            "type": "sidebar",
            "sidebar_id": "medusa-cli",
            "title": "Medusa CLI",
            "childSidebarTitle": "Medusa CLI Reference",
            "initialOpen": true,
            "children": []
          }
        ]
      },
      {
        "loaded": true,
        "type": "category",
        "title": "Developer Tools",
        "initialOpen": true,
        "description": "Developer tools facilitate the development of your Medusa application by providing utilities for linting and other uses.",
        "children": [
          {
            "loaded": true,
            "type": "sidebar",
            "sidebar_id": "lint",
            "title": "ESLint Plugin",
            "childSidebarTitle": "ESLint Plugin Reference",
            "initialOpen": true,
            "children": []
          }
        ]
      },
      {
        "loaded": true,
        "type": "category",
        "title": "SDKs",
        "initialOpen": true,
        "description": "SDKs help you build client applications, such as storefronts or admin dashboards, with Medusa. They're also useful when extending the Medusa Admin with widgets and UI routes.",
        "children": [
          {
            "loaded": true,
            "type": "sidebar",
            "sidebar_id": "js-sdk",
            "title": "JS SDK",
            "childSidebarTitle": "JS SDK Reference",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/js-sdk",
                "title": "Overview",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/js-sdk/auth/overview",
                "title": "Authentication",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "sidebar",
            "sidebar_id": "admin-components",
            "title": "Medusa Admin Components",
            "childSidebarTitle": "Admin Components Reference",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/admin-components",
                "title": "Overview",
                "children": []
              },
              {
                "type": "separator"
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Components",
                "children": [
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/components",
                    "title": "Overview",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/components/action-menu",
                    "title": "Action Menu",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/components/combobox",
                    "title": "Combobox",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/components/conditional-tooltip",
                    "title": "Conditional Tooltip",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/components/configurable-data-table",
                    "title": "Configurable Data Table",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/components/country-select",
                    "title": "Country Select",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/components/data-grid",
                    "title": "Data Grid",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/components/display-id",
                    "title": "Display ID",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/components/empty-states",
                    "title": "Empty States",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/components/handle-input",
                    "title": "Handle Input",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/components/icon-avatar",
                    "title": "Icon Avatar",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/components/json-view-section",
                    "title": "JSON View Section",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/components/keybound-form",
                    "title": "Keybound Form",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/components/listicle",
                    "title": "Listicle",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/components/product-table-cells",
                    "title": "Product Table Cells",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/components/route-drawer",
                    "title": "Route Drawer",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/components/route-focus-modal",
                    "title": "Route Focus Modal",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/components/section-row",
                    "title": "Section Row",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/components/stacked-drawer",
                    "title": "Stacked Drawer",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/components/stacked-focus-modal",
                    "title": "Stacked Focus Modal",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/components/thumbnail",
                    "title": "Thumbnail",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Layouts",
                "autogenerate_path": "/admin-components/layouts",
                "children": [
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/layouts/layout-composer",
                    "title": "LayoutComposer",
                    "description": "",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Hooks",
                "children": [
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/hooks",
                    "title": "Overview",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/hooks/customers",
                    "title": "Customers",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/hooks/data-table-helpers",
                    "title": "Data Table Hooks",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/hooks/orders",
                    "title": "Orders",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/hooks/price-preferences",
                    "title": "Price Preferences",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/hooks/product-variants",
                    "title": "Product Variants",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/hooks/products",
                    "title": "Products",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/hooks/promotions",
                    "title": "Promotions",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/hooks/regions",
                    "title": "Regions",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/hooks/sales-channels",
                    "title": "Sales Channels",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/hooks/shipping-options",
                    "title": "Shipping Options",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/hooks/store",
                    "title": "Store",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/hooks/use-combobox-data",
                    "title": "useComboboxData",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/hooks/use-date",
                    "title": "useDate",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/hooks/use-debounced-search",
                    "title": "useDebouncedSearch",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/hooks/use-query-params",
                    "title": "useQueryParams",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/hooks/users",
                    "title": "Users",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Utilities",
                "children": [
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/utilities",
                    "title": "Overview",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/utilities/addresses",
                    "title": "Addresses",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/utilities/form-validation",
                    "title": "Form Validation",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/utilities/money-and-currency",
                    "title": "Money & Currency",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/utilities/search",
                    "title": "Search Entities",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/utilities/table",
                    "title": "Table Utilities",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Guides",
                "autogenerate_path": "/admin-components/guides",
                "children": [
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/guides/forms",
                    "title": "Forms",
                    "description": "",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/guides/container",
                    "title": "Container",
                    "description": "",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/guides/table",
                    "title": "Table",
                    "description": "",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/guides/header",
                    "title": "Header",
                    "description": "",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/guides/data-table",
                    "title": "Data Table",
                    "description": "",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Archive",
                "children": [
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/archive",
                    "title": "Overview",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/archive/action-menu",
                    "title": "Action Menu",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/archive/json-view-section",
                    "title": "JSON View",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/archive/section-row",
                    "title": "Section Row",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/archive/single-column-layout",
                    "title": "Single Column Layout",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/admin-components/archive/two-column-layout",
                    "title": "Two Column Layout",
                    "children": []
                  }
                ]
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "category",
        "title": "Storefront Starters",
        "initialOpen": true,
        "description": "A storefront starter is a storefront with the basic commerce features that you can use with your Medusa application. You can use it as-is or build on top of it. Learn more about building a storefront in the [Storefront Development](/storefront-development) documentation.",
        "children": [
          {
            "loaded": true,
            "type": "sidebar",
            "sidebar_id": "nextjs-starter",
            "title": "Next.js Starter Storefront",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/nextjs-starter",
                "title": "Overview",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "How-to Guides",
                "initialOpen": true,
                "children": [
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/nextjs-starter/guides/revalidate-cache",
                    "title": "Revalidate Cache",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "link",
                    "path": "/resources/nextjs-starter/guides/remove-country-code",
                    "title": "Remove Country Code from URLs",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Tutorials",
                "autogenerate_tags": "nextjs+tutorial",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              }
            ]
          }
        ]
      }
    ]
  },
  {
    "sidebar_id": "references",
    "title": "References",
    "items": [
      {
        "loaded": true,
        "type": "link",
        "path": "/resources/admin-widget-injection-zones",
        "title": "Admin Widget Injection Zones",
        "description": "List of injection zones you can add widgets in.",
        "children": []
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "dml-reference",
        "title": "Data Model Language",
        "childSidebarTitle": "Data Model Language Reference",
        "description": "API reference of the Data Model Langauge (DML).",
        "children": []
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "data-model-repository-reference",
        "title": "Data Model Repository",
        "description": "List of methods in your service extending the data model repository.",
        "children": []
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "helper-steps-reference",
        "title": "Helper Steps",
        "description": "Steps to query data, emit events, create links, and more.",
        "children": []
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "service-factory-reference",
        "title": "Service Factory",
        "description": "List of methods in your service extending the service factory.",
        "children": []
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "test-tools-reference",
        "title": "Testing Framework",
        "description": "API reference of functions you can use to write tests.",
        "children": []
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "workflows-sdk-reference",
        "title": "Workflows SDK",
        "childSidebarTitle": "Workflows SDK Reference",
        "description": "API reference of the Workflow SDK functions and helpers.",
        "children": []
      }
    ]
  },
  {
    "sidebar_id": "commerce-modules",
    "title": "Commerce Modules",
    "items": [
      {
        "loaded": true,
        "type": "link",
        "title": "Overview",
        "path": "/resources/commerce-modules",
        "children": []
      },
      {
        "type": "separator"
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "api-key",
        "title": "API Key Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/api-key",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Concepts",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/api-key/concepts",
                "title": "API Key Concepts",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/api-key/links-to-other-modules",
                "title": "Link to Modules",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Storefront Guides",
            "autogenerate_tags": "storefront+apiKey,-jsSdk",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to integrate the API Key Module's features into your storefront.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Use a Publishable API Key in the Storefront",
                "path": "/resources/storefront-development/publishable-api-keys",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin User Guides",
            "autogenerate_tags": "userGuide+apiKey",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize and manage API Key features in the Medusa Admin dashboard.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Publishable API Keys",
                "path": "/user-guide/settings/developer/publishable-api-keys",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Secret API Keys",
                "path": "/user-guide/settings/developer/secret-api-keys",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "References",
            "description": "Find references for tools and resources related to the API Key Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "category",
                "title": "Workflows",
                "autogenerate_tags": "workflow+apiKey",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Steps",
                "autogenerate_tags": "step+apiKey",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/api-key/js-sdk",
                "title": "JS SDK",
                "hideChildren": true,
                "children": [
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Admin",
                    "autogenerate_tags": "jsSdk+admin+apiKey",
                    "description": "The following methods or properties are used to send requests to Admin API Routes related to the API Key Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/api-key/admin-widget-zones",
                "title": "Admin Widget Zones",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "api-key-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "API Key Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "api-key-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "API Key Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "auth",
        "title": "Auth Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/auth",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Concepts",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/auth/auth-identity-and-actor-types",
                "title": "Identity and Actor Types",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/auth/auth-flows",
                "title": "Auth Flow with Module",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/auth/authentication-route",
                "title": "Auth Flow with Routes",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/auth/mfa",
                "title": "Multi-Factor Authentication",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Server Guides",
            "autogenerate_tags": "server+auth",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to use the Auth Module in your customizations on the Medusa application server.",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/auth/create-actor-type",
                "title": "Create Actor Type",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/auth/reset-password",
                "title": "Handle Password Reset Event",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/auth/link-provider-identity",
                "title": "Link Provider to Existing Actor",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Restrict Order Retrieval",
                "path": "/resources/commerce-modules/order/secure-order-retrieval",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Restrict Return Creation",
                "path": "/resources/commerce-modules/order/secure-return-creation",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/auth/email-verification",
                "title": "Send Email Verification",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Storefront Guides",
            "autogenerate_tags": "storefront+auth,-jsSdk",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to integrate the Auth Module's features into your storefront.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Log-out Customer in Storefront",
                "path": "/resources/storefront-development/customers/log-out",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Login Customer in Storefront",
                "path": "/resources/storefront-development/customers/login",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Register Customer in Storefront",
                "path": "/resources/storefront-development/customers/register",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Register Customer with Email Verification in Storefront",
                "path": "/resources/storefront-development/customers/verify-account",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Reset Customer Password in Storefront",
                "path": "/resources/storefront-development/customers/reset-password",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Retrieve Logged-In Customer in Storefront",
                "path": "/resources/storefront-development/customers/retrieve",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Third-Party or Social Login in Storefront",
                "path": "/resources/storefront-development/customers/third-party-login",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin User Guides",
            "autogenerate_tags": "userGuide+auth",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize and manage Auth features in the Medusa Admin dashboard.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Login Options",
                "path": "/user-guide/tips/login-options",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Reset Password",
                "path": "/user-guide/reset-password",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Providers",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/auth/auth-providers/google",
                "title": "Google Provider",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/auth/auth-providers/github",
                "title": "GitHub Provider",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "References",
            "description": "Find references for tools and resources related to the Auth Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "category",
                "title": "Workflows",
                "autogenerate_tags": "workflow+auth",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Steps",
                "autogenerate_tags": "step+auth",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/auth/js-sdk",
                "title": "JS SDK",
                "hideChildren": true,
                "autogenerate_tags": "jsSdk+auth",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/auth/admin-widget-zones",
                "title": "Admin Widget Zones",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "auth-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "Auth Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "auth-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "Auth Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "cart",
        "title": "Cart Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/cart",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Concepts",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/cart/concepts",
                "title": "Cart Concepts",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/cart/promotions",
                "title": "Promotion Adjustments",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/cart/tax-lines",
                "title": "Tax Lines",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/cart/links-to-other-modules",
                "title": "Links to Other Modules",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Server Guides",
            "autogenerate_tags": "cart+server",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to use the Cart Module in your customizations on the Medusa application server.",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/cart/sales-channel-availability",
                "title": "Enforce Sales Channel Availability",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/cart/cart-totals",
                "title": "Retrieve Cart Totals",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Storefront Guides",
            "autogenerate_tags": "storefront+cart,-jsSdk",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to integrate the Cart Module's features into your storefront.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Checkout Step 1: Enter Email",
                "path": "/resources/storefront-development/checkout/email",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Checkout Step 2: Set Address",
                "path": "/resources/storefront-development/checkout/address",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Checkout Step 3: Choose Shipping Method",
                "path": "/resources/storefront-development/checkout/shipping",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Checkout Step 4: Choose Payment Provider",
                "path": "/resources/storefront-development/checkout/payment",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Checkout Step 5: Complete Cart",
                "path": "/resources/storefront-development/checkout/complete-cart",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Create Cart Context in Storefront",
                "path": "/resources/storefront-development/cart/context",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Create Cart in Storefront",
                "path": "/resources/storefront-development/cart/create",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Implement Express Checkout with Medusa",
                "path": "/resources/storefront-development/guides/express-checkout",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Implement Mobile App with React Native, Expo, and Medusa",
                "path": "/resources/storefront-development/guides/react-native-expo",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Cart Promotions in Storefront",
                "path": "/resources/storefront-development/cart/manage-promotions",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Cart's Items in Storefront",
                "path": "/resources/storefront-development/cart/manage-items",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Payment with Stripe in React Storefront",
                "path": "/resources/storefront-development/checkout/payment/stripe",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Retrieve Cart in Storefront",
                "path": "/resources/storefront-development/cart/retrieve",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Show Cart Totals",
                "path": "/resources/storefront-development/cart/totals",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Update Cart in Storefront",
                "path": "/resources/storefront-development/cart/update",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "References",
            "description": "Find references for tools and resources related to the Cart Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "category",
                "title": "Workflows",
                "autogenerate_tags": "workflow+cart",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Steps",
                "autogenerate_tags": "step+cart",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/cart/js-sdk",
                "title": "JS SDK",
                "hideChildren": true,
                "children": [
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Store",
                    "autogenerate_tags": "jsSdk+storefront+cart",
                    "description": "The following methods or properties are used to send requests to Store API Routes related to the Cart Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "cart-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "Cart Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "cart-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "Cart Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "currency",
        "title": "Currency Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/currency",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "sub-category",
            "title": "Concepts",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/currency/links-to-other-modules",
                "title": "Link to Modules",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin User Guides",
            "autogenerate_tags": "userGuide+currency",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize and manage Currency features in the Medusa Admin dashboard.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Store",
                "path": "/user-guide/settings/store",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "sub-category",
            "title": "References",
            "description": "Find references for tools and resources related to the Currency Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/currency/js-sdk",
                "title": "JS SDK",
                "hideChildren": true,
                "children": [
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Admin",
                    "autogenerate_tags": "jsSdk+admin+currency",
                    "description": "The following methods or properties are used to send requests to Admin API Routes related to the Currency Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "currency-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "Currency Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "currency-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "Currency Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "customer",
        "title": "Customer Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/customer",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Concepts",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/customer/customer-accounts",
                "title": "Customer Accounts",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/customer/links-to-other-modules",
                "title": "Link to Modules",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Server Guides",
            "autogenerate_tags": "customer+server",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to use the Customer Module in your customizations on the Medusa application server.",
            "children": []
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Storefront Guides",
            "autogenerate_tags": "storefront+customer,-jsSdk",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to integrate the Customer Module's features into your storefront.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Customer Context in Storefront",
                "path": "/resources/storefront-development/customers/context",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Edit Customer Profile in Storefront",
                "path": "/resources/storefront-development/customers/profile",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Log-out Customer in Storefront",
                "path": "/resources/storefront-development/customers/log-out",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Login Customer in Storefront",
                "path": "/resources/storefront-development/customers/login",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Customer Addresses in Storefront",
                "path": "/resources/storefront-development/customers/addresses",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Register Customer in Storefront",
                "path": "/resources/storefront-development/customers/register",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Register Customer with Email Verification in Storefront",
                "path": "/resources/storefront-development/customers/verify-account",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Reset Customer Password in Storefront",
                "path": "/resources/storefront-development/customers/reset-password",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Retrieve Logged-In Customer in Storefront",
                "path": "/resources/storefront-development/customers/retrieve",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Third-Party or Social Login in Storefront",
                "path": "/resources/storefront-development/customers/third-party-login",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin User Guides",
            "autogenerate_tags": "userGuide+customer",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize and manage Customer features in the Medusa Admin dashboard.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Customers Overview",
                "path": "/user-guide/customers",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Customer Groups",
                "path": "/user-guide/customers/groups",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Customers",
                "path": "/user-guide/customers/manage",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "References",
            "description": "Find references for tools and resources related to the Customer Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "category",
                "title": "Workflows",
                "autogenerate_tags": "workflow+customer",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Steps",
                "autogenerate_tags": "step+customer",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/customer/js-sdk",
                "title": "JS SDK",
                "hideChildren": true,
                "children": [
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Store",
                    "autogenerate_tags": "jsSdk+storefront+customer",
                    "description": "The following methods or properties are used to send requests to Store API Routes related to the Customer Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Admin",
                    "autogenerate_tags": "jsSdk+admin+customer",
                    "description": "The following methods or properties are used to send requests to Admin API Routes related to the Customer Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/customer/admin-widget-zones",
                "title": "Admin Widget Zones",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "customer-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "Customer Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "customer-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "Customer Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "fulfillment",
        "title": "Fulfillment Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/fulfillment",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Concepts",
            "autogenerate_tags": "fulfillment+concept",
            "autogenerate_as_ref": true,
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/fulfillment/concepts",
                "title": "General Concepts",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/fulfillment/item-fulfillment",
                "title": "Item Fulfillment",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/fulfillment/fulfillment-provider",
                "title": "Fulfillment Module Provider",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/fulfillment/shipping-option",
                "title": "Shipping Option",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/fulfillment/links-to-other-modules",
                "title": "Links to Other Modules",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Reservations Lifecycle",
                "path": "/resources/commerce-modules/inventory/reservations-lifecycle",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Shipping Option Price Rules",
                "path": "/resources/commerce-modules/pricing/price-rules",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Physical and Customs Attributes",
                "path": "/resources/commerce-modules/product/physical-customs-attributes",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Product Shipping Requirement",
                "path": "/resources/commerce-modules/product/selling-products",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Server Guides",
            "autogenerate_tags": "server+fulfillment",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to use the Fulfillment Module in your customizations on the Medusa application server.",
            "children": []
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Storefront Guides",
            "autogenerate_tags": "storefront+fulfillment,-jsSdk",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to integrate the Fulfillment Module's features into your storefront.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Checkout Step 3: Choose Shipping Method",
                "path": "/resources/storefront-development/checkout/shipping",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Implement Express Checkout with Medusa",
                "path": "/resources/storefront-development/guides/express-checkout",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Implement Mobile App with React Native, Expo, and Medusa",
                "path": "/resources/storefront-development/guides/react-native-expo",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin User Guides",
            "autogenerate_tags": "fulfillment+userGuide",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize and manage Fulfillment features in the Medusa Admin dashboard.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Locations & Shipping Overview",
                "path": "/user-guide/settings/locations-and-shipping",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Order Fulfillments",
                "path": "/user-guide/orders/fulfillments",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Shipping Option Types",
                "path": "/user-guide/settings/locations-and-shipping/shipping-option-types",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Shipping Options",
                "path": "/user-guide/settings/locations-and-shipping/locations",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Shipping Profiles",
                "path": "/user-guide/settings/locations-and-shipping/shipping-profiles",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "References",
            "description": "Find references for tools and resources related to the Fulfillment Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "category",
                "title": "Workflows",
                "autogenerate_tags": "workflow+fulfillment",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Steps",
                "autogenerate_tags": "step+fulfillment",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/fulfillment/js-sdk",
                "title": "JS SDK",
                "hideChildren": true,
                "children": [
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Store",
                    "autogenerate_tags": "jsSdk+storefront+fulfillment",
                    "description": "The following methods or properties are used to send requests to Store API Routes related to the Fulfillment Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Admin",
                    "autogenerate_tags": "jsSdk+admin+fulfillment",
                    "description": "The following methods or properties are used to send requests to Admin API Routes related to the Fulfillment Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/fulfillment/admin-widget-zones",
                "title": "Admin Widget Zones",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "fulfillment-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "Fulfillment Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "fulfillment-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "Fulfillment Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "inventory",
        "title": "Inventory Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/inventory",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Concepts",
            "autogenerate_tags": "concept+inventory",
            "autogenerate_as_ref": true,
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/inventory/concepts",
                "title": "Inventory Concepts",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/inventory/inventory-in-flows",
                "title": "Inventory in Flows",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/inventory/reservations-lifecycle",
                "title": "Reservations Lifecycle",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/inventory/inventory-kit",
                "title": "Inventory Kit",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/inventory/links-to-other-modules",
                "title": "Links to Modules",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Physical and Customs Attributes",
                "path": "/resources/commerce-modules/product/physical-customs-attributes",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Product Variant Inventory",
                "path": "/resources/commerce-modules/product/variant-inventory",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Server Guides",
            "autogenerate_tags": "server+inventory",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to use the Inventory Module in your customizations on the Medusa application server.",
            "children": []
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Storefront Guides",
            "autogenerate_tags": "storefront+inventory,-jsSdk",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to integrate the Inventory Module's features into your storefront.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Retrieve Product Variant's Inventory in Storefront",
                "path": "/resources/storefront-development/products/inventory",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin User Guides",
            "autogenerate_tags": "userGuide+inventory",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize and manage Inventory features in the Medusa Admin dashboard.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Export Inventory Items",
                "path": "/user-guide/inventory/export",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Inventory Overview",
                "path": "/user-guide/inventory",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Inventory Items",
                "path": "/user-guide/inventory/inventory",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Reservations",
                "path": "/user-guide/inventory/reservations",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "References",
            "description": "Find references for tools and resources related to the Inventory Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "category",
                "title": "Workflows",
                "autogenerate_tags": "workflow+inventory",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Steps",
                "autogenerate_tags": "step+inventory",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/inventory/js-sdk",
                "title": "JS SDK",
                "hideChildren": true,
                "children": [
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Admin",
                    "autogenerate_tags": "jsSdk+admin+inventory",
                    "description": "The following methods or properties are used to send requests to Admin API Routes related to the Inventory Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/inventory/admin-widget-zones",
                "title": "Admin Widget Zones",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "inventory-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "Inventory Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "inventory-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "Inventory Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "loyalty",
        "title": "Loyalty Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/loyalty",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Concepts",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/loyalty/gift-cards",
                "title": "Gift Cards",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/loyalty/links-to-other-modules",
                "title": "Link to Modules",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin User Guides",
            "autogenerate_tags": "userGuide+loyalty",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize and manage Loyalty features in the Medusa Admin dashboard.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Gift Card Products Overview",
                "path": "/user-guide/loyalty/gift-card-products",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Gift Cards Overview",
                "path": "/user-guide/loyalty/gift-cards",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Loyalty Features in Medusa Admin",
                "path": "/user-guide/loyalty",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Gift Card Products",
                "path": "/user-guide/loyalty/gift-card-products/manage",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Gift Cards",
                "path": "/user-guide/loyalty/gift-cards/manage",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "References",
            "description": "Find references for tools and resources related to the Loyalty Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "category",
                "title": "Workflows",
                "autogenerate_tags": "workflow+loyalty",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Steps",
                "autogenerate_tags": "step+loyalty",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/loyalty/admin-widget-zones",
                "title": "Admin Widget Zones",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "loyalty-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "Loyalty Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "loyalty-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "Loyalty Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "order",
        "title": "Order Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/order",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Concepts",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/order/concepts",
                "title": "Order Concepts",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/order/custom-display-id",
                "title": "Custom Order Display ID",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/order/promotion-adjustments",
                "title": "Promotions Adjustments",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/order/tax-lines",
                "title": "Tax Lines",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/order/transactions",
                "title": "Transactions",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/order/order-versioning",
                "title": "Order Versioning",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/order/draft-orders",
                "title": "Draft Orders",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/order/return",
                "title": "Return",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/order/exchange",
                "title": "Exchange",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/order/claim",
                "title": "Claim",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/order/edit",
                "title": "Order Edit",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/order/order-change",
                "title": "Order Change",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/order/transfer-to-guest",
                "title": "Transfer Order to Guest",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/order/links-to-other-modules",
                "title": "Links to Other Modules",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Server Guides",
            "autogenerate_tags": "order+server",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to use the Order Module in your customizations on the Medusa application server.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Restrict Order Retrieval",
                "path": "/resources/commerce-modules/order/secure-order-retrieval",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Restrict Return Creation",
                "path": "/resources/commerce-modules/order/secure-return-creation",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/order/order-totals",
                "title": "Retrieve Order Totals",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Storefront Guides",
            "autogenerate_tags": "storefront+order,-jsSdk",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to integrate the Order Module's features into your storefront.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Checkout Step 5: Complete Cart",
                "path": "/resources/storefront-development/checkout/complete-cart",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Implement Express Checkout with Medusa",
                "path": "/resources/storefront-development/guides/express-checkout",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Implement Mobile App with React Native, Expo, and Medusa",
                "path": "/resources/storefront-development/guides/react-native-expo",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Order Confirmation in Storefront",
                "path": "/resources/storefront-development/checkout/order-confirmation",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin User Guides",
            "autogenerate_tags": "userGuide+order",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize and manage Order features in the Medusa Admin dashboard.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Create Draft Order",
                "path": "/user-guide/orders/draft-orders/create",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Draft Orders",
                "path": "/user-guide/orders/draft-orders",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Edit Order Items",
                "path": "/user-guide/orders/edit",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Export Orders",
                "path": "/user-guide/orders/export",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Draft Order",
                "path": "/user-guide/orders/draft-orders/manage",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Order Claims",
                "path": "/user-guide/orders/claims",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Order Details",
                "path": "/user-guide/orders/manage",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Order Exchanges",
                "path": "/user-guide/orders/exchanges",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Order Fulfillments",
                "path": "/user-guide/orders/fulfillments",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Order Payments",
                "path": "/user-guide/orders/payments",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Order Returns",
                "path": "/user-guide/orders/returns",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Refund Reasons",
                "path": "/user-guide/settings/refund-reasons",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Return Reasons",
                "path": "/user-guide/settings/return-reasons",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Orders Overview",
                "path": "/user-guide/orders",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "References",
            "description": "Find references for tools and resources related to the Order Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "category",
                "title": "Workflows",
                "autogenerate_tags": "workflow+order",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Steps",
                "autogenerate_tags": "step+order",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/order/js-sdk",
                "title": "JS SDK",
                "hideChildren": true,
                "children": [
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Store",
                    "autogenerate_tags": "jsSdk+storefront+order",
                    "description": "The following methods or properties are used to send requests to Store API Routes related to the Order Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Admin",
                    "autogenerate_tags": "jsSdk+admin+order",
                    "description": "The following methods or properties are used to send requests to Admin API Routes related to the Order Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/order/admin-widget-zones",
                "title": "Admin Widget Zones",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "order-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "Order Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "order-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "Order Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "payment",
        "title": "Payment Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/payment",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Concepts",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/payment/payment-collection",
                "title": "Payment Collections",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/payment/payment-session",
                "title": "Payment Session",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/payment/payment",
                "title": "Payment",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/payment/payment-provider",
                "title": "Payment Module Provider",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/payment/payment-checkout-flow",
                "title": "Payment in Checkout",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/payment/account-holder",
                "title": "Account Holder",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/payment/webhook-events",
                "title": "Webhook Events",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/payment/links-to-other-modules",
                "title": "Links to Other Modules",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Server Guides",
            "autogenerate_tags": "server+payment",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to use the Payment Module in your customizations on the Medusa application server.",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/payment/payment-flow",
                "title": "Accept Payment Flow",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Storefront Guides",
            "autogenerate_tags": "storefront+payment,-jsSdk",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to integrate the Payment Module's features into your storefront.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Checkout Step 4: Choose Payment Provider",
                "path": "/resources/storefront-development/checkout/payment",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Checkout Step 5: Complete Cart",
                "path": "/resources/storefront-development/checkout/complete-cart",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Implement Express Checkout with Medusa",
                "path": "/resources/storefront-development/guides/express-checkout",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Implement Mobile App with React Native, Expo, and Medusa",
                "path": "/resources/storefront-development/guides/react-native-expo",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Payment with Stripe in React Storefront",
                "path": "/resources/storefront-development/checkout/payment/stripe",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin User Guides",
            "autogenerate_tags": "userGuide+payment",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize and manage Payment features in the Medusa Admin dashboard.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Order Payments",
                "path": "/user-guide/orders/payments",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Providers",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/payment/payment-provider/stripe",
                "title": "Stripe",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/payment/payment-provider/adyen",
                "title": "Adyen",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "References",
            "description": "Find references for tools and resources related to the Payment Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "category",
                "title": "Workflows",
                "autogenerate_tags": "workflow+payment",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Steps",
                "autogenerate_tags": "step+payment",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/payment/js-sdk",
                "title": "JS SDK",
                "hideChildren": true,
                "children": [
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Store",
                    "autogenerate_tags": "jsSdk+storefront+payment",
                    "description": "The following methods or properties are used to send requests to Store API Routes related to the Payment Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Admin",
                    "autogenerate_tags": "jsSdk+admin+payment",
                    "description": "The following methods or properties are used to send requests to Admin API Routes related to the Payment Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "payment-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "Payment Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "payment-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "Payment Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "pricing",
        "title": "Pricing Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/pricing",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Concepts",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/pricing/concepts",
                "title": "Pricing Concepts",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/pricing/price-rules",
                "title": "Price Tiers and Rules",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/pricing/price-calculation",
                "title": "Prices Calculation",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/pricing/tax-inclusive-pricing",
                "title": "Tax-Inclusive Pricing",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/pricing/links-to-other-modules",
                "title": "Links to Other Modules",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Server Guides",
            "autogenerate_tags": "server+pricing",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to use the Pricing Module in your customizations on the Medusa application server.",
            "children": []
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Storefront Guides",
            "autogenerate_tags": "storefront+pricing,-jsSdk",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to integrate the Pricing Module's features into your storefront.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Example: Show Price with Taxes",
                "path": "/resources/storefront-development/products/price/examples/tax-price",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Example: Show Sale Price",
                "path": "/resources/storefront-development/products/price/examples/sale-price",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Example: Show Variant's Price",
                "path": "/resources/storefront-development/products/price/examples/show-price",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Retrieve Product Variant's Prices in Storefront",
                "path": "/resources/storefront-development/products/price",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin User Guides",
            "autogenerate_tags": "userGuide+pricing",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize and manage Pricing features in the Medusa Admin dashboard.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Create a Price List",
                "path": "/user-guide/price-lists/create",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Price Lists",
                "path": "/user-guide/price-lists/manage",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Price Lists Overview",
                "path": "/user-guide/price-lists",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "References",
            "description": "Find references for tools and resources related to the Pricing Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "category",
                "title": "Workflows",
                "autogenerate_tags": "workflow+pricing",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Steps",
                "autogenerate_tags": "step+pricing",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/pricing/js-sdk",
                "title": "JS SDK",
                "hideChildren": true,
                "children": [
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Admin",
                    "autogenerate_tags": "jsSdk+admin+pricing",
                    "description": "The following methods or properties are used to send requests to Admin API Routes related to the Pricing Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/pricing/admin-widget-zones",
                "title": "Admin Widget Zones",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "pricing-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "Pricing Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "pricing-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "Pricing Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "product",
        "title": "Product Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/product",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Concepts",
            "autogenerate_tags": "product+concept",
            "autogenerate_as_ref": true,
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/product/variant-inventory",
                "title": "Variant Inventory",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/product/physical-customs-attributes",
                "title": "Physical and Customs Attributes",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/product/selling-products",
                "title": "Selling Use Cases",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/product/links-to-other-modules",
                "title": "Links to Other Modules",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Inventory Kits",
                "path": "/resources/commerce-modules/inventory/inventory-kit",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Variant Price Rules",
                "path": "/resources/commerce-modules/pricing/price-rules",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Server Guides",
            "autogenerate_tags": "product+server",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to use the Product Module in your customizations on the Medusa application server.",
            "children": []
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Storefront Guides",
            "autogenerate_tags": "storefront+product,-jsSdk",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to integrate the Product Module's features into your storefront.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Example: Show Price with Taxes",
                "path": "/resources/storefront-development/products/price/examples/tax-price",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Example: Show Sale Price",
                "path": "/resources/storefront-development/products/price/examples/sale-price",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Example: Show Variant's Price",
                "path": "/resources/storefront-development/products/price/examples/show-price",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Implement Express Checkout with Medusa",
                "path": "/resources/storefront-development/guides/express-checkout",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Implement Mobile App with React Native, Expo, and Medusa",
                "path": "/resources/storefront-development/guides/react-native-expo",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "List Product Collections in Storefront",
                "path": "/resources/storefront-development/products/collections/list",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Retrieve a Category in Storefront",
                "path": "/resources/storefront-development/products/categories/retrieve",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Retrieve a Category's Products in Storefront",
                "path": "/resources/storefront-development/products/categories/products",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Retrieve a Collection in Storefront",
                "path": "/resources/storefront-development/products/collections/retrieve",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Retrieve a Collection's Products in Storefront",
                "path": "/resources/storefront-development/products/collections/products",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Retrieve a Product in Storefront",
                "path": "/resources/storefront-development/products/retrieve",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Retrieve Nested Categories in Storefront",
                "path": "/resources/storefront-development/products/categories/nested-categories",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Retrieve Product Variant's Inventory in Storefront",
                "path": "/resources/storefront-development/products/inventory",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Retrieve Product Variant's Prices in Storefront",
                "path": "/resources/storefront-development/products/price",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Select Product Variants in Storefront",
                "path": "/resources/storefront-development/products/variants",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Show Product Categories in Storefront",
                "path": "/resources/storefront-development/products/categories/list",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Show Products in Storefront",
                "path": "/resources/storefront-development/products/list",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Use Product Options in Storefront",
                "path": "/resources/storefront-development/products/options",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin Guides",
            "autogenerate_tags": "admin+product,-jsSdk",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize administative features of the Product Module.",
            "children": []
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin User Guides",
            "autogenerate_tags": "userGuide+product",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize and manage Product features in the Medusa Admin dashboard.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Create Bundle Product",
                "path": "/user-guide/products/create/bundle",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Create Multi-Part Product",
                "path": "/user-guide/products/create/multi-part",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Create Product",
                "path": "/user-guide/products/create",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Edit Product",
                "path": "/user-guide/products/edit",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Export Products",
                "path": "/user-guide/products/export",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Import Products",
                "path": "/user-guide/products/import",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Product Categories",
                "path": "/user-guide/products/categories",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Product Collections",
                "path": "/user-guide/products/collections",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Product Options",
                "path": "/user-guide/products/options",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Product Tags",
                "path": "/user-guide/settings/product-tags",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Product Types",
                "path": "/user-guide/settings/product-types",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Product Variants",
                "path": "/user-guide/products/variants",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Products Overview",
                "path": "/user-guide/products",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "References",
            "description": "Find references for tools and resources related to the Product Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "category",
                "title": "Workflows",
                "autogenerate_tags": "workflow+product",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Steps",
                "autogenerate_tags": "step+product",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/product/js-sdk",
                "title": "JS SDK",
                "hideChildren": true,
                "children": [
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Store",
                    "autogenerate_tags": "jsSdk+storefront+product",
                    "description": "The following methods or properties are used to send requests to Store API Routes related to the Product Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Admin",
                    "autogenerate_tags": "jsSdk+admin+product",
                    "description": "The following methods or properties are used to send requests to Admin API Routes related to the Product Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/product/admin-widget-zones",
                "title": "Admin Widget Zones",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "product-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "Product Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "product-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "Product Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "promotion",
        "title": "Promotion Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/promotion",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Concepts",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/promotion/concepts",
                "title": "Promotion",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/promotion/promotion-taxes",
                "title": "Tax-Inclusive Promotions",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/promotion/application-method",
                "title": "Application Method",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/promotion/campaign",
                "title": "Campaign",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/promotion/actions",
                "title": "Promotion Actions",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/promotion/links-to-other-modules",
                "title": "Links to Modules",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Server Guides",
            "autogenerate_tags": "promotion+server",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to use the Promotion Module in your customizations on the Medusa application server.",
            "children": []
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Storefront Guides",
            "autogenerate_tags": "storefront+promotion,-jsSdk",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to integrate the Promotion Module's features into your storefront.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Cart Promotions in Storefront",
                "path": "/resources/storefront-development/cart/manage-promotions",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin User Guides",
            "autogenerate_tags": "userGuide+promotion",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize and manage Promotion features in the Medusa Admin dashboard.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Create a Promotion",
                "path": "/user-guide/promotions/create",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Campaigns",
                "path": "/user-guide/promotions/campaigns",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Promotions",
                "path": "/user-guide/promotions/manage",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Promotions Overview",
                "path": "/user-guide/promotions",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "References",
            "description": "Find references for tools and resources related to the Promotion Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "category",
                "title": "Workflows",
                "autogenerate_tags": "workflow+promotion",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Steps",
                "autogenerate_tags": "step+promotion",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/promotion/js-sdk",
                "title": "JS SDK",
                "hideChildren": true,
                "children": [
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Admin",
                    "autogenerate_tags": "jsSdk+admin+promotion",
                    "description": "The following methods or properties are used to send requests to Admin API Routes related to the Promotion Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/promotion/admin-widget-zones",
                "title": "Admin Widget Zones",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "promotion-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "Promotion Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "promotion-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "Promotion Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "region",
        "title": "Region Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/region",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Concepts",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/region/links-to-other-modules",
                "title": "Links to Modules",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Storefront Guides",
            "autogenerate_tags": "storefront+region,-jsSdk",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to integrate the Region Module's features into your storefront.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Implement Express Checkout with Medusa",
                "path": "/resources/storefront-development/guides/express-checkout",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Implement Mobile App with React Native, Expo, and Medusa",
                "path": "/resources/storefront-development/guides/react-native-expo",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "List Regions in Storefront",
                "path": "/resources/storefront-development/regions/list",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Region React Context in Storefront",
                "path": "/resources/storefront-development/regions/context",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Store Selected Region in Storefront",
                "path": "/resources/storefront-development/regions/store-retrieve-region",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin User Guides",
            "autogenerate_tags": "userGuide+region",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize and manage Region features in the Medusa Admin dashboard.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Regions",
                "path": "/user-guide/settings/regions",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "References",
            "description": "Find references for tools and resources related to the Region Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "category",
                "title": "Workflows",
                "autogenerate_tags": "workflow+region",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Steps",
                "autogenerate_tags": "step+region",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/region/js-sdk",
                "title": "JS SDK",
                "hideChildren": true,
                "children": [
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Store",
                    "autogenerate_tags": "jsSdk+storefront+region",
                    "description": "The following methods or properties are used to send requests to Store API Routes related to the Region Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  },
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Admin",
                    "autogenerate_tags": "jsSdk+admin+region",
                    "description": "The following methods or properties are used to send requests to Admin API Routes related to the Region Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/region/admin-widget-zones",
                "title": "Admin Widget Zones",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "region-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "Region Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "region-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "Region Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "sales-channel",
        "title": "Sales Channel Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/sales-channel",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Concepts",
            "autogenerate_tags": "concept+salesChannel",
            "autogenerate_as_ref": true,
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/sales-channel/publishable-api-keys",
                "title": "Publishable API Keys",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/sales-channel/links-to-other-modules",
                "title": "Links to Modules",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Product Variant Inventory",
                "path": "/resources/commerce-modules/product/variant-inventory",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Server Guides",
            "autogenerate_tags": "server+salesChannel",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to use the Sales Channel Module in your customizations on the Medusa application server.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Enforce Sales Channel Availability",
                "path": "/resources/commerce-modules/cart/sales-channel-availability",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Storefront Guides",
            "autogenerate_tags": "storefront+salesChannel,-jsSdk",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to integrate the Sales Channel Module's features into your storefront.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Use a Publishable API Key in the Storefront",
                "path": "/resources/storefront-development/publishable-api-keys",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin User Guides",
            "autogenerate_tags": "userGuide+salesChannel",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize and manage Sales Channel features in the Medusa Admin dashboard.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Sales Channels",
                "path": "/user-guide/settings/sales-channels",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "References",
            "description": "Find references for tools and resources related to the Sales Channel Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "category",
                "title": "Workflows",
                "autogenerate_tags": "workflow+salesChannel",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Steps",
                "autogenerate_tags": "step+salesChannel",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/sales-channel/js-sdk",
                "title": "JS SDK",
                "hideChildren": true,
                "children": [
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Admin",
                    "autogenerate_tags": "jsSdk+admin+salesChannel",
                    "description": "The following methods or properties are used to send requests to Admin API Routes related to the Sales Channel Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/sales-channel/admin-widget-zones",
                "title": "Admin Widget Zones",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "sales-channel-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "Sales Channel Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "sales-channel-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "Sales Channel Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "settings",
        "title": "Settings Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/settings",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Concepts",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/settings/concepts",
                "title": "Concepts",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/settings/view-configurations",
                "title": "View Configurations",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/settings/layout-configurations",
                "title": "Layout Configurations",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/settings/property-labels",
                "title": "Property Labels",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/settings/user-preferences",
                "title": "User Preferences",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Server Guides",
            "autogenerate_tags": "server+settings",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to use the Settings Module in your customizations on the Medusa application server.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Configure View Configurations",
                "path": "/resources/commerce-modules/settings/configure-view-configurations",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin Guides",
            "autogenerate_tags": "admin+settings,-jsSdk",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize administrative features of the Settings Module.",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/settings/configure-view-configurations",
                "title": "Configure View Configurations",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin User Guides",
            "autogenerate_tags": "userGuide+settings",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize and manage Settings features in the Medusa Admin dashboard.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Configure Layouts",
                "path": "/user-guide/tips/layout-configurations",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Configure Table Views",
                "path": "/user-guide/tips/table-views",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Property Labels",
                "path": "/user-guide/settings/property-labels",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "References",
            "description": "Find references for tools and resources related to the Settings Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "category",
                "title": "Workflows",
                "autogenerate_tags": "workflow+settings",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Steps",
                "autogenerate_tags": "step+settings",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/settings/js-sdk",
                "title": "JS SDK",
                "hideChildren": true,
                "children": [
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Admin",
                    "autogenerate_tags": "jsSdk+admin+settings,jsSdk+admin+views",
                    "description": "The following methods or properties are used to send requests to Admin API Routes related to the Settings Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/settings/admin-widget-zones",
                "title": "Admin Widget Zones",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "settings-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "Settings Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "settings-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "Settings Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "stock-location",
        "title": "Stock Location Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/stock-location",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Concepts",
            "autogenerate_tags": "concept+stockLocation",
            "autogenerate_as_ref": true,
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/stock-location/concepts",
                "title": "Stock Location Concepts",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/stock-location/links-to-other-modules",
                "title": "Links to Modules",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Inventory Kits",
                "path": "/resources/commerce-modules/inventory/inventory-kit",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Product Variant Inventory",
                "path": "/resources/commerce-modules/product/variant-inventory",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin User Guides",
            "autogenerate_tags": "userGuide+stockLocation",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize and manage Stock Location features in the Medusa Admin dashboard.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Locations & Shipping Overview",
                "path": "/user-guide/settings/locations-and-shipping",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Locations",
                "path": "/user-guide/settings/locations-and-shipping/locations",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "References",
            "description": "Find references for tools and resources related to the Stock Location Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "category",
                "title": "Workflows",
                "autogenerate_tags": "workflow+stockLocation",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Steps",
                "autogenerate_tags": "step+stockLocation",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/stock-location/js-sdk",
                "title": "JS SDK",
                "hideChildren": true,
                "children": [
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Admin",
                    "autogenerate_tags": "jsSdk+admin+stockLocation",
                    "description": "The following methods or properties are used to send requests to Admin API Routes related to the Stock Location Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/stock-location/admin-widget-zones",
                "title": "Admin Widget Zones",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "stock-location-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "Stock Location Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "stock-location-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "Stock Location Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "store",
        "title": "Store Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/store",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Concepts",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/store/locales",
                "title": "Locales",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/store/links-to-other-modules",
                "title": "Link to Modules",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin User Guides",
            "autogenerate_tags": "userGuide+store",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize and manage Store features in the Medusa Admin dashboard.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Store",
                "path": "/user-guide/settings/store",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "References",
            "description": "Find references for tools and resources related to the Store Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "category",
                "title": "Workflows",
                "autogenerate_tags": "workflow+store",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Steps",
                "autogenerate_tags": "step+store",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/store/js-sdk",
                "title": "JS SDK",
                "hideChildren": true,
                "children": [
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Admin",
                    "autogenerate_tags": "jsSdk+admin+store",
                    "description": "The following methods or properties are used to send requests to Admin API Routes related to the Store Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/store/admin-widget-zones",
                "title": "Admin Widget Zones",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "store-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "Store Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "store-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "Store Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "store-credit",
        "title": "Store Credit Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/store-credit",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Concepts",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/store-credit/concepts",
                "title": "Concepts",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/store-credit/links-to-other-modules",
                "title": "Link to Modules",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin User Guides",
            "autogenerate_tags": "userGuide+storeCredit",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize and manage Store Credit features in the Medusa Admin dashboard.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Store Credits",
                "path": "/user-guide/loyalty/store-credits/manage",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Store Credits Overview",
                "path": "/user-guide/loyalty/store-credits",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "References",
            "description": "Find references for tools and resources related to the Store Credit Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "category",
                "title": "Workflows",
                "autogenerate_tags": "workflow+storeCredit",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Steps",
                "autogenerate_tags": "step+storeCredit",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/store-credit/admin-widget-zones",
                "title": "Admin Widget Zones",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "store-credit-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "Store Credit Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "store-credit-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "Store Credit Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "tax",
        "title": "Tax Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/tax",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Concepts",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/tax/tax-region",
                "title": "Tax Region",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/tax/tax-rates-and-rules",
                "title": "Tax Rates and Rules",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/tax/tax-provider",
                "title": "Tax Module Providers",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Server Guides",
            "autogenerate_tags": "server+tax",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to use the Tax Module in your customizations on the Medusa application server.",
            "children": []
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Storefront Guides",
            "autogenerate_tags": "storefront+tax,-jsSdk",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to integrate the Tax Module's features into your storefront.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Example: Show Price with Taxes",
                "path": "/resources/storefront-development/products/price/examples/tax-price",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin User Guides",
            "autogenerate_tags": "userGuide+tax",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize and manage Tax features in the Medusa Admin dashboard.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Tax Regions",
                "path": "/user-guide/settings/tax-regions",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "References",
            "description": "Find references for tools and resources related to the Tax Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "category",
                "title": "Workflows",
                "autogenerate_tags": "workflow+tax",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Steps",
                "autogenerate_tags": "step+tax",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/tax/js-sdk",
                "title": "JS SDK",
                "hideChildren": true,
                "children": [
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Admin",
                    "autogenerate_tags": "jsSdk+admin+tax",
                    "description": "The following methods or properties are used to send requests to Admin API Routes related to the Tax Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/tax/admin-widget-zones",
                "title": "Admin Widget Zones",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "tax-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "Tax Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "tax-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "Tax Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "translation",
        "title": "Translation Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/translation",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Concepts",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/translation/concepts",
                "title": "Concepts",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/translation/custom-data-models",
                "title": "Translate Custom Models",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/translation/storefront",
                "title": "Translation in Storefront",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/translation/links-to-other-modules",
                "title": "Link to Modules",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Storefront Guides",
            "autogenerate_tags": "storefront+translation,-jsSdk",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to integrate the Translation Module's features into your storefront.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Storefront Localization",
                "path": "/resources/storefront-development/localization",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin User Guides",
            "autogenerate_tags": "userGuide+translation",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize and manage Translation features in the Medusa Admin dashboard.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Store",
                "path": "/user-guide/settings/store",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Translations",
                "path": "/user-guide/settings/translations",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "References",
            "description": "Find references for tools and resources related to the Translation Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "category",
                "title": "Workflows",
                "autogenerate_tags": "workflow+translation",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Steps",
                "autogenerate_tags": "step+translation",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/translation/js-sdk",
                "title": "JS SDK",
                "hideChildren": true,
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "translation-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "Translation Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "translation-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "Translation Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      },
      {
        "loaded": true,
        "type": "sidebar",
        "sidebar_id": "user",
        "title": "User Module",
        "children": [
          {
            "loaded": true,
            "type": "link",
            "path": "/resources/commerce-modules/user",
            "title": "Overview",
            "children": []
          },
          {
            "type": "separator"
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Concepts",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/user/links-to-other-modules",
                "title": "Link to Modules",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Server Guides",
            "autogenerate_tags": "server+user",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to use the User Module in your customizations on the Medusa application server.",
            "children": [
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/user/invite-user-subscriber",
                "title": "Invite User Email",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "Admin User Guides",
            "autogenerate_tags": "userGuide+user",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "description": "Learn how to utilize and manage User features in the Medusa Admin dashboard.",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Login Options",
                "path": "/user-guide/tips/login-options",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Invites",
                "path": "/user-guide/settings/users/invites",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Profile",
                "path": "/user-guide/settings/profile",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Manage Users",
                "path": "/user-guide/settings/users",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Reset Password",
                "path": "/user-guide/reset-password",
                "children": []
              }
            ]
          },
          {
            "loaded": true,
            "type": "category",
            "title": "References",
            "description": "Find references for tools and resources related to the User Module, such as data models, methods, and more. These are useful for your customizations.",
            "children": [
              {
                "loaded": true,
                "type": "category",
                "title": "Workflows",
                "autogenerate_tags": "workflow+user",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "category",
                "title": "Steps",
                "autogenerate_tags": "step+user",
                "autogenerate_as_ref": true,
                "sort_sidebar": "alphabetize",
                "children": []
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/user/js-sdk",
                "title": "JS SDK",
                "hideChildren": true,
                "children": [
                  {
                    "loaded": true,
                    "type": "sub-category",
                    "title": "Admin",
                    "autogenerate_tags": "jsSdk+admin+user",
                    "description": "The following methods or properties are used to send requests to Admin API Routes related to the User Module.",
                    "autogenerate_as_ref": true,
                    "sort_sidebar": "alphabetize",
                    "children": []
                  }
                ]
              },
              {
                "loaded": true,
                "type": "link",
                "path": "/resources/commerce-modules/user/admin-widget-zones",
                "title": "Admin Widget Zones",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "user-service-reference",
                "title": "Main Service Reference",
                "childSidebarTitle": "User Module's Main Service Reference",
                "children": []
              },
              {
                "loaded": true,
                "type": "sidebar",
                "sidebar_id": "user-models-reference",
                "title": "Data Models Reference",
                "childSidebarTitle": "User Module Data Models Reference",
                "children": []
              }
            ]
          }
        ]
      }
    ]
  },
  {
    "sidebar_id": "infrastructure-modules",
    "title": "Infrastructure Modules",
    "items": [
      {
        "loaded": true,
        "type": "category",
        "title": "Notification Module",
        "initialOpen": true,
        "children": [
          {
            "loaded": true,
            "type": "sub-category",
            "title": "Guides",
            "autogenerate_tags": "notification+server",
            "autogenerate_as_ref": true,
            "sort_sidebar": "alphabetize",
            "children": [
              {
                "loaded": true,
                "type": "ref",
                "title": "Invite User Email",
                "path": "/resources/commerce-modules/user/invite-user-subscriber",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Reset Password Email",
                "path": "/resources/commerce-modules/auth/reset-password",
                "children": []
              },
              {
                "loaded": true,
                "type": "ref",
                "title": "Send Email Verification",
                "path": "/resources/commerce-modules/auth/email-verification",
                "children": []
              }
            ]
          }
        ]
      }
    ]
  },
  {
    "sidebar_id": "troubleshooting",
    "title": "Troubleshooting",
    "items": []
  }
]

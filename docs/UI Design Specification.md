## ErpMate UI Design Specification

Prepared for the UI developer. Based on the two Horizon EBS reference screens: the module launcher (home) and the transaction form (Quotation Details).

All sizes and colors below are measured by eye from the screenshots. Treat them as the target and confirm against the screenshots at 100% zoom before sign-off.

## 1. Design principles

- 1. Clean and flat. White cards on a light gray page. Soft shadows, no gradients, no decorative art inside the app.

- 2. One brand color plus one action color. A dark primary color for the frame (top bar, sidebar) and one accent for actions, active states and key labels. Everything else is neutral gray or white.

- 3. Color means something. Each business module has its own color. That color appears only on that module's icon, number, progress bar and button.

- 4. Filled inputs, not outlined. Fields have a light tinted background with a bottom line. The active field is marked by the accent color on its bottom line.

- 5. Dense but readable. Data-entry screens use a tight grid and small text (12 to 13px). Launcher screens use more space.

- 6. Consistent building blocks. Every screen is built from the same few components: top bar, sidebar, toolbar, card, field, button, table. Do not invent new variants.

- 7. Font: Poppins for the whole app (weights 400, 500, 600, 700).

## 2. Theme tokens

Build every color as a CSS variable so the whole app can be re-themed in one place. The reference uses navy and red. ErpMate uses green and white, so both are listed.

| Token | Reference | ErpMate | Used for |
| --- | --- | --- | --- |
|   | (Horizon) | (green) |   |
| --color-primary | #0B2A5B | #0A3324 | Top bar, sidebar |
| --color-primary-active | #071E45 | #06241A | Active sidebar item |
| --color-accent | #E53935 | #14663F | Primary buttons, active tab line, breadcrumbs, section highlights |
| --color-accent-hover | #C62828 | #0F4A33 | Button hover |\
| --color-accent-tint | #FDECEA | #E6F1EB | Highlighted input fill, selected row |
| --color-page-bg | #F2F3F5 | #F2F5F3 | Page background |
| --color-surface | #FFFFFF | #FFFFFF | Cards, table |
| --color-field-bg | #F1F0F7 | #EEF5F1 | Input background |
| --color-field-line | #C9C9D6 | #B9CBC1 | Input bottom line |
| --color-text | #1E2430 | #0F2A1E | Main text |
| --color-text-muted | #6B7280 | #5B6A62 | Placeholders, captions |
| --color-border | #E5E7EB | #E4EBE7 | Card and table borders |
| --color-success | #2E9E5B | #2E9E5B | Save icon, online dot |
| --color-danger | #D32F2F | #B42318 | Cancel icon, errors |

## Module colors (launcher cards)

These stay the same in both themes so each module is easy to recognize.

| Module | Color |
| --- | --- |
| Inventory Management | #D81B60 |
| Human Resources | #7CB342 |
| Sales Distribution | #FB8C00 |
| Finance & Accounting | #00A67E |


| Module | Color |
| --- | --- |
| Project Accounting | #29B6F6 |
| Bank Facility | #FDD835 |
| Fixed Asset Management | #7B3FA0 |
| Customer Relationship | #E53935 |

Each module also needs a light tint of its color (about 25% opacity) for the unfilled part of its progress bar.

## 3. Typography

| Role | Size / weight | Example |
| --- | --- | --- |
| Page greeting | 18px / 400 | "Welcome Merina Adam" (name in accent color) |
| Card title (form section) | 16px / 600 | "Quotation Details" |
| Module card title | 14px / 600 | "Inventory Management" |
| Module metric | 24px / 600, module color| "5000", "$30,000" |
| Body and field text | 13px / 400 | Inputs, table cells |
| Field label and placeholder| 12px / 400, muted | "Quotation Numbers*" |
| Table header | 12px / 600 | "Prod. Code" |
| Caption | 11px / 400, muted | "50% increase in 30 days" |
| Button text | 11 to 12px / 600 | "Get Started", "Save" |
| Totals | 13px / 600, accent color | "Gross Amount (in Rs): 750000" |

## 4. Spacing, radius and shadow

- Spacing scale: 4, 8, 12, 16, 24, 32 px. Use only these.

- Card padding: 16 to 24px.

- Gap between form fields: 8px horizontally, 12px vertically.

- Gap between launcher cards: 20px.

- Radius: cards 8px, buttons 4px, inputs 4px on top corners only, pills and avatars fully round.

- Shadow (cards): 0 2px 8px rgba(16, 24, 40, 0.08) . Use the same shadow on every card.

- Shadow (primary buttons): 0 2px 6px in a darker tint of the button color.

## 5. Application shell

Every screen except login uses the same shell.

## 5.1 Top bar

- Height 48px, background --color-primary , fixed at the top.

- Left: logo (white), then a hamburger icon that collapses the sidebar.

- Center-left: a white search box with a search icon, 8px radius, about 360px wide, placeholder "Search...".

- Right, in order: + Add New button (accent, 4px radius), notification bell with a small accent badge showing the count, user name with a dropdown arrow, round 28px avatar.

- Text on the bar is white, 12px.

## 5.2 Sidebar

- Width about 140px (collapsed: icons only, 56px). Background --color-primary .

- Top block: round 40px avatar, user name (600), and a small green "Online" dot with label.

- Menu items: icon (16px) plus label (13px, white). Item height 40px, left padding 16px.

- Active item: darker background --color-primary-active , full width.

- Items with children show a small down arrow on the right.

- A thin divider separates the main menu from the bottom group (Archive/Trash, Log Out).


- Menu items shown: Dashboard, Masters, Listing, Enquiry, Sales Transactions, Sales Report. Use your own modules for ErpMate (Procurement, Inventory, Production, Sales, Reports).

## 5.3 Page header on form screens

- Toolbar under the top bar: a row of icon buttons with a small label under each icon (Add, Edit, Delete, Refresh, Save, Search, Find, Cancel, Post, Print, Exit). Icon 18px, label 10px.

- Disabled actions are light gray.

- Save uses the success color, Cancel uses the danger color.

- Only actions that make sense for the current state are enabled.

- Record navigation on the right of the toolbar: a joined button group First | Previous | Showing 1 of 9584 | Next | Last , 12px text, 1px border, white background.

- Breadcrumb below the toolbar: accent-colored text separated by > , 12px.

## 6. Screen A: Module launcher (home)

Purpose: choose a company and year, then open a module.

- Top bar: logo on the left. On the right: "Welcome" followed by the user's name in the accent color, then the avatar with a dropdown. No sidebar on this screen.

- Page background: --color-page-bg , content max width about 1100px, centered.

- Filter row: two large selects side by side, "Select Company*" and "Years*".

- White background, 1px accent border, 4px radius, height 44px, search icon on the right, muted placeholder.

- Card grid: 4 columns by 2 rows, 20px gap. On narrow screens: 2 columns, then 1.

- Module card (white, 8px radius, shared shadow, 16px padding), top to bottom:

- 1. Icon badge: 28px rounded square (6px radius) filled with the module color, white icon inside.

- 2. Title: 14px, 600.

- 3. Metric: 24px, 600, in the module color.

- 4. Progress bar: 3px high, rounded. The first third is the solid module color, the rest is its light tint.

- 5. Caption: 11px muted, for example "50% increase in 30 days".


- 6. Get Started button: small, solid module color, white 10px text, 3px radius, left aligned, about 56 x 20px.

- The whole card is clickable. Hover: the shadow gets slightly stronger. No movement or scaling.

## 7. Screen B: Transaction form (Quotation Details)

Purpose: create or edit one document with a header section and a line-items table.

## 7.1 Header card

- White card, full width, 8px radius, shared shadow. Title "Quotation Details" at the top left.

- Two columns inside the card, separated by a 1px vertical divider.

- Left column (document and customer details): a grid of fields, 2 to 3 per row. Examples: Quotation Number, Date, Customer ID, Email, Address, City & Country, Contact Person, Reference, Project, Consultant, Client, Validity Period.

- Right column (commercial details): Currency, Exchange Rate, Sales Location, Salesperson, Payment Terms, Terms, and a larger Remarks field.

- Status group: a heading "Status" in the accent color and bold, with radio buttons (In Process, Approved, Lost, Void).

- Quotation Type group: same heading style with radios (Tender, Job in Hand).

- Actions at the bottom right: a small "Del. Details" button, and a full-width accent button "Show Last 10 Transactions (Press F1)".

## 7.2 Product details card

- Title "Quotation Product Details" at the left. Small read-only fields on the right (SIC, Base UOM, QOH, SO Qty, Packing).

- Tabs: "Product Details" and "Clause (Terms & Conditions) Details". Inside a rounded bordered container. The active tab has a 2px accent underline and bold text.

## Table:

- Columns: Line No, Product Code, Description, UOM, Qty, Total Qty, Rate, Disc %, Amount, Print Details (checkbox), Columns menu, row actions (kebab icon).

- Header row: light background, 12px, 600. Rows: 40px high, 1px bottom border, no vertical lines.

- Numbers are right aligned. Text is left aligned.


- The "Columns" control at the right of the header lets the user show or hide columns.

- Body scrolls vertically. The header stays fixed.

## 7.3 Footer bar (sticky)

- Fixed to the bottom of the content area, white, with a top border.

- Left: Disc % checkbox, Disc % input and Disc Amount input. These two inputs use -- color-accent-tint as their fill to show they are editable totals.

- Right: Gross Amount and Net Amount, bold, accent color, right aligned, with the currency in brackets.

## 8. Components

## 8.1 Input (filled field)

- Height 40px, background --color-field-bg , no border except a 2px bottom line -- color-field-line , radius 4px 4px 0 0, 12px horizontal padding, 13px text.

- Label: sits inside as the placeholder. When the field is focused or has a value, the label moves to the top as a 9px floating label.

- Required: a red asterisk after the label text.

- Lookup field: a small search icon on the right. Clicking it opens a search dialog.

- Focus: the bottom line turns accent color. No glow and no outer border.

- Error: bottom line in danger color and a 11px error message below.

- Disabled or read-only: same shape, text in muted color, no hover.


## 8.2 Buttons

| Type | Style |
| --- | --- |
| Primary | Accent fill, white 12px 600 text, 4px radius, height 32 to 44px depending on context |
| Module action | Same shape, filled with the module color |
| Toolbar | Icon with label under it, no background, disabled = light gray |
| Segmented (pagination)|  Joined group with shared 1px border, white fill |   |

Rules: one primary button per section. Hover darkens the fill. Disabled buttons use 50% opacity and show no pointer.

## 8.3 Radio buttons and checkboxes

- 16px, accent color when selected, 12px label to the right.

- Radios in a group are laid out horizontally with 16px spacing.

## 8.4 Card

- White, 8px radius, shared shadow, 16 to 24px padding. A card has a title and optionally a right-aligned toolbar. Do not nest cards inside cards.

## 8.5 Table

- As described in section 7.2. Row hover uses --color-accent-tint at low opacity. Selected row uses the full tint.

## 8.6 Tabs

- Horizontal, equal width inside a bordered, rounded container. Active: bold text plus a 2px accent underline.

## 9. Behavior

- Sidebar: the hamburger toggles between full and icon-only.


- Toolbar state: Save is enabled only when the form has unsaved changes. Cancel discards changes after a confirm dialog. Edit and Delete are disabled while the form is already in edit mode.

- Validation: required fields are checked on Save. The first invalid field gets focus and the error style.

- Keyboard: Tab follows reading order. F1 opens "Show Last 10 Transactions". Enter in a lookup field opens the search dialog.

- Loading: show a thin progress bar under the top bar, and a spinner inside the button that was clicked.

- Empty table: one centered line of muted text and an "Add line" action.

## 10. Responsive and accessibility

- Breakpoints: 1280px and above is the full layout. 768 to 1279px collapses the sidebar to icons and the header card to a single column. Below 768px the sidebar becomes a drawer and the launcher shows one card per row.

- Tables scroll horizontally inside their own container. The page itself must never scroll sideways.

- Contrast: body text must meet WCAG AA (4.5 to 1). White text on the yellow Bank Facility button fails, so use dark text on that one button.

- Focus: every interactive element has a visible focus ring (3px, accent at 40% opacity).

- Icons need accessible labels. Do not rely on color alone for status.

- Respect prefers-reduced-motion .

## 11. Suggested implementation

Stack: React, Vite and TypeScript, as in the current frontend.


```
src/
  styles/
    tokens.css                /* all CSS variables from section 2 */
    base.css                 /* font, reset, body */
  shared/components/
    AppShell/               /* TopBar, Sidebar */
    Toolbar/
    Card/
    Field/                  /* filled input, lookup, floating label */
    Button/
    DataTable/
    Tabs/
  modules/
    launcher/               /* Screen A */
    sales/quotation/     /* Screen B */
```

- Use plain CSS with variables or CSS modules. Do not hard-code any color from section 2 inside a component.

- Build Field , Button , Card , DataTable first, then assemble both screens from them.

- Use one icon set (for example Lucide) at 16 to 18px, stroke width 2.

- Load Poppins from Google Fonts or self-host it.

## 12. Acceptance checklist

- Only the tokens in section 2 are used for colors.

- Font is Poppins everywhere.

- Every card has the same radius and shadow.

- Inputs are filled with a bottom line, and focus changes only the line color.

- Launcher shows 4 by 2 cards with the correct module colors.

- Toolbar, breadcrumb and record navigation appear on every form screen.

- Table header stays fixed and the footer totals bar stays visible.

- Layout works at 1280px, 1024px and 375px widths.

- Keyboard focus is visible and Tab order is correct.

- Side-by-side comparison with the reference screens shows no visible differences in spacing, color or type.

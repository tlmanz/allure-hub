# Design System Specification



## 1. Overview & Creative North Star



### Creative North Star: "The Obsidian Architect"

The design system is engineered for the high-stakes world of technical observability and complex data analysis. It moves beyond the typical "flat" dashboard by embracing **The Obsidian Architect**—a philosophy that treats the UI as a dense, high-performance instrument carved out of deep, layered mineral tones.



Instead of a generic grid, this system utilizes **intentional asymmetry** and **tonal depth** to guide the eye. We break the "template" look by layering surfaces like sheets of dark glass, using high-contrast typography scales (Space Grotesk vs. Inter) to differentiate between "The Story" (high-level metrics) and "The Data" (functional details). The result is an editorial-grade technical experience that feels authoritative, premium, and calm under pressure.



---



## 2. Colors & Atmospheric Depth



This palette is designed to maintain high legibility while reducing eye strain during extended technical sessions.



### Core Palette

* **Background (`#040e1f`):** The foundational "Obsidian" base. All depth builds from here.

* **Primary (`#5cfd80`):** Vibrant neon green for positive health, success states, and primary actions.

* **Secondary (`#1db1f1`):** Electric blue for information and interactive components.

* **Tertiary/Warning (`#ff8762`):** A sophisticated burnt orange for caution and mid-tier alerts.

* **Error (`#ff716c`):** A high-visibility coral-red for critical failures.



### The "No-Line" Rule

**Explicit Instruction:** Traditional 1px solid borders for sectioning are strictly prohibited. The UI must feel like a single, cohesive ecosystem. Boundaries are defined solely through:

1. **Background Shifts:** Using `surface-container-low` against `surface`.

2. **Vertical Space:** Utilizing the Spacing Scale (specifically `8` to `12`) to separate logical groups.



### Surface Hierarchy & Glassmorphism

We treat the UI as physical layers.

* **Nesting:** Place a `surface-container-highest` (`#15263f`) card inside a `surface-container-low` (`#061326`) section to create natural focus.

* **Glass & Gradient Rule:** For floating modals or dropdowns, use `surface-bright` with a 60% opacity and a `20px` backdrop-blur. Apply a subtle linear gradient from `primary` to `primary_container` on mission-critical CTAs to give them "soul."



---



## 3. Typography: Editorial Functionality



We use a tri-font strategy to balance character with data density.



* **Display & Headlines (Space Grotesk):** This is our "Editorial" voice. It is geometric and modern. Use `display-lg` (3.5rem) for hero metrics to give the dashboard an authoritative, high-end feel.

* **Titles & Body (Inter):** Our "Functional" voice. Inter’s tall x-height ensures readability for complex logs. `body-md` (0.875rem) in `on_surface_variant` (`#a0abc2`) provides a soft, light-gray contrast that is legible without being harsh.

* **Labels (Manrope):** Our "Technical" voice. Used for small data labels (`label-sm`, 0.6875rem). Manrope’s condensed nature allows for dense data visualization without clutter.



---



## 4. Elevation & Depth: Tonal Layering



Shadows and borders are secondary to color-blocking. We achieve depth through the **Layering Principle**.



* **Tonal Stacking:** To lift a card, do not reach for a shadow first. Change the token from `surface` to `surface_container`.

* **Ambient Shadows:** For floating elements (menus, tooltips), use a hyper-diffused shadow: `box-shadow: 0 20px 40px rgba(0, 0, 0, 0.4)`. The shadow must feel like ambient light blockage, not a "drop shadow."

* **The Ghost Border:** If high-density data requires containment, use the `outline_variant` token at **15% opacity**. This creates a "Ghost Border" that provides structure without breaking the seamless obsidian look.



---



## 5. Components



### Buttons & Chips

* **Primary Button:** `primary` background with `on_primary` text. Use `lg` (0.5rem) rounded corners.

* **Status Chips:** Use a "Glass-Fill" approach. A `success` chip should have a 10% opacity `primary` background and a 100% opaque `primary` text label. This mimics the reference image's treatment of status indicators.



### Cards & Lists

* **Card Structure:** No dividers. Use `surface_container_high` and a padding of `8` (1.75rem).

* **Asymmetric Data:** Don't center-align everything. Use the "Editorial" layout: Large display metrics on the right, supporting technical meta-data on the left.



### Technical Dashboards (Custom)

* **The "Pulse" Indicator:** Use a soft radial gradient on the `primary` color with a CSS scale animation to show live system health.

* **Ring Charts:** As seen in the reference, use high-stroke-width circles (approx 8-10px) with `surface_variant` as the track color and `primary`/`tertiary`/`error` for the data segments.



---



## 6. Do's and Don'ts



### Do:

* **DO** use `surface-container-lowest` (`#000000`) for the most recessed areas of the app (e.g., a code editor or log terminal).

* **DO** use the `16` (3.5rem) spacing token between major dashboard modules to allow the layout to breathe.

* **DO** tint your shadows with the background hue to maintain the dark-mode richness.



### Don't:

* **DON'T** use pure white (`#FFFFFF`) for body text. Use `on_surface_variant` (`#a0abc2`) to prevent "halation" (text glowing) on dark backgrounds.

* **DON'T** use `none` or `sm` corners for main cards. The system requires the "Premium" feel of `md` (0.375rem) or `lg` (0.5rem) radiuses.

* **DON'T** use 100% opaque borders to separate list items. Use a 2px vertical gap (spacing `0.5`) to let the background bleed through.
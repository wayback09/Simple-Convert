---
name: Monochrome Precision
colors:
  surface: '#131313'
  surface-dim: '#131313'
  surface-bright: '#393939'
  surface-container-lowest: '#0e0e0e'
  surface-container-low: '#1b1b1b'
  surface-container: '#1f1f1f'
  surface-container-high: '#2a2a2a'
  surface-container-highest: '#353535'
  on-surface: '#e2e2e2'
  on-surface-variant: '#c4c7c8'
  inverse-surface: '#e2e2e2'
  inverse-on-surface: '#303030'
  outline: '#8e9192'
  outline-variant: '#444748'
  surface-tint: '#c6c6c7'
  primary: '#ffffff'
  on-primary: '#2f3131'
  primary-container: '#e2e2e2'
  on-primary-container: '#636565'
  inverse-primary: '#5d5f5f'
  secondary: '#c8c6c5'
  on-secondary: '#313030'
  secondary-container: '#474746'
  on-secondary-container: '#b7b5b4'
  tertiary: '#ffffff'
  on-tertiary: '#2f3131'
  tertiary-container: '#e2e2e2'
  on-tertiary-container: '#636565'
  error: '#ffb4ab'
  on-error: '#690005'
  error-container: '#93000a'
  on-error-container: '#ffdad6'
  primary-fixed: '#e2e2e2'
  primary-fixed-dim: '#c6c6c7'
  on-primary-fixed: '#1a1c1c'
  on-primary-fixed-variant: '#454747'
  secondary-fixed: '#e5e2e1'
  secondary-fixed-dim: '#c8c6c5'
  on-secondary-fixed: '#1c1b1b'
  on-secondary-fixed-variant: '#474746'
  tertiary-fixed: '#e2e2e2'
  tertiary-fixed-dim: '#c6c6c7'
  on-tertiary-fixed: '#1a1c1c'
  on-tertiary-fixed-variant: '#454747'
  background: '#131313'
  on-background: '#e2e2e2'
  surface-variant: '#353535'
typography:
  headline-lg:
    fontFamily: Inter
    fontSize: 48px
    fontWeight: '700'
    lineHeight: 56px
    letterSpacing: -0.02em
  headline-lg-mobile:
    fontFamily: Inter
    fontSize: 32px
    fontWeight: '700'
    lineHeight: 40px
    letterSpacing: -0.01em
  headline-md:
    fontFamily: Inter
    fontSize: 24px
    fontWeight: '600'
    lineHeight: 32px
    letterSpacing: -0.01em
  body-lg:
    fontFamily: Inter
    fontSize: 18px
    fontWeight: '400'
    lineHeight: 28px
  body-md:
    fontFamily: Inter
    fontSize: 16px
    fontWeight: '400'
    lineHeight: 24px
  body-sm:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 20px
  label-md:
    fontFamily: Inter
    fontSize: 12px
    fontWeight: '600'
    lineHeight: 16px
    letterSpacing: 0.05em
rounded:
  sm: 0.125rem
  DEFAULT: 0.25rem
  md: 0.375rem
  lg: 0.5rem
  xl: 0.75rem
  full: 9999px
spacing:
  base: 4px
  xs: 8px
  sm: 16px
  md: 24px
  lg: 40px
  xl: 64px
  gutter: 24px
  margin: 32px
---

## Brand & Style
The design system moves away from cybernetic vibrance toward a strict, high-contrast aesthetic rooted in **Minimalism** and **Modern Brutalism**. The personality is clinical, authoritative, and focused, stripping away chromatic distractions to emphasize structure and content. It targets professional environments where clarity and precision are paramount.

The UI evokes an emotional response of absolute stability and technical sophistication. By utilizing a binary color logic (pure black and pure white), the interface establishes an uncompromising hierarchy. The visual language is defined by sharp edges, intentional whitespace, and a complete absence of hue.

## Colors
The palette is strictly monochromatic, utilizing a dark-mode default to maintain a "terminal" or "studio" atmosphere.

- **Primary Background:** Pure black (#000000) for the deepest level of the interface.
- **Surface/Container:** Deep near-black gray (#0A0A0A) to create subtle separation between layered elements.
- **Accents:** Pure white (#FFFFFF) is reserved exclusively for primary actions, critical highlights, and high-level headings.
- **Borders:** Mid-to-dark grays (#333333) define boundaries without breaking the high-contrast immersion.
- **Interactive States:** Hover and active states utilize inverted values—white backgrounds with black text for maximum impact.

## Typography
This design system utilizes **Inter** across all levels to maintain a systematic, utilitarian aesthetic. 

- **Headlines:** Set in pure white (#FFFFFF) with tight letter spacing and bold weights to create a commanding presence.
- **Body Text:** Set in a slightly dimmed white or light gray (#E5E5E5) for readability, reducing eye strain against the black background.
- **Secondary/Metadata:** Set in mid-gray (#A1A1A1) to recede in the visual hierarchy.
- **Labels:** Small-caps or all-caps styling is used for technical labels to reinforce the "precision instrument" feel.

## Layout & Spacing
The layout follows a rigorous 12-column fluid grid for desktop and a single-column stack for mobile. Spacing is based on a 4px/8px modular scale to ensure technical alignment.

- **Desktop:** 12 columns, 24px gutters, 40px+ side margins.
- **Tablet:** 8 columns, 16px gutters, 24px side margins.
- **Mobile:** 4 columns, 16px gutters, 16px side margins.

Content is grouped using generous whitespace rather than excessive lines, creating a sense of "breathable density." Alignment is strictly flush-left to emphasize the vertical rhythm of the typography.

## Elevation & Depth
Depth is communicated through **Tonal Layering** and **Low-Contrast Outlines** rather than traditional shadows. 

1. **Level 0 (Floor):** Pure black (#000000).
2. **Level 1 (Card/Container):** Near-black (#0A0A0A) with a subtle 1px border (#333333).
3. **Level 2 (Popovers/Modals):** Dark gray (#141414) with a slightly brighter border (#444444).

Shadows, if used, are sharp and opaque ("hard shadows"), offset by 4px or 8px to mimic physical layering without using blurs. This reinforces the brutalist nature of the design system.

## Shapes
In alignment with the "ROUND_FOUR" requirement, the design system utilizes a **Soft** shape language. This subtle rounding provides a necessary bridge between the harshness of the monochrome palette and a modern, high-end feel.

- **Standard Components:** 0.25rem (4px) corner radius.
- **Large Containers/Cards:** 0.5rem (8px) corner radius.
- **Buttons:** Consistently 4px rounded to maintain a rigid, structural appearance.

## Components

- **Buttons:** 
  - *Primary:* Pure white background, black text, bold weight.
  - *Secondary:* Transparent background, 1px white border, white text.
  - *States:* Hovering on primary triggers a slight gray shift (#E5E5E5); hovering on secondary fills the background with white.
- **Input Fields:** 
  - Black background with a 1px gray (#333333) border. On focus, the border turns pure white. Placeholder text is mid-gray.
- **Cards:** 
  - Subtle #0A0A0A background. Borders are mandatory for cards to distinguish them from the floor background.
- **Chips/Labels:** 
  - Small, rectangular, with 4px radius. Use inverted color schemes (White background/Black text) for "Active" or "Success" states to ensure they stand out without using color.
- **Lists:** 
  - Separated by thin 1px horizontal rules (#222222). Items utilize high-contrast white for titles and gray for descriptions.
- **Checkboxes/Radios:** 
  - Pure white borders. When checked, the inner glyph is pure white against the black background.
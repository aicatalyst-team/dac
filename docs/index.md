---
layout: home

hero:
  name: DAC
  text: Dashboard-as-Code
  tagline: Define, validate, and serve dashboards from YAML and TSX. Embedded frontend, Bruin-powered query execution.
  actions:
    - theme: brand
      text: Get Started
      link: /getting-started/installation
    - theme: alt
      text: View on GitHub
      link: https://github.com/bruin-data/dac

features:
  - title: YAML & TSX
    details: Define dashboards in plain YAML for simplicity, or TSX for full programmatic control with loops, variables, and load-time queries.
  - title: 17 Chart Types
    details: Line, bar, area, pie, scatter, bubble, combo, histogram, boxplot, funnel, sankey, heatmap, calendar, sparkline, waterfall, XMR, and dumbbell.
  - title: Semantic Layer
    details: Declare metrics and dimensions once, reference them across widgets. DAC generates the SQL for you.
  - title: Jinja Templating
    details: Dynamic SQL with filter variables, conditionals, and loops. Queries react to user-selected filters in real time.
  - title: Live Reload
    details: Edit your YAML or TSX, save, and see changes instantly in the browser. No restart needed.
  - title: Single Binary
    details: Go backend with embedded React frontend. DAC ships as one binary, while query execution uses the Bruin connections you already manage.
  - title: Static Export
    details: Build self-contained HTML dashboards with baked-in query results. Deploy anywhere — no server required.
  - title: Google Slides Export
    details: Export dashboards to Google Slides presentations with charts rendered as images and data baked in.
---

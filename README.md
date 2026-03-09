# SML Pages MVP (WordPress Plugin)

MVP plugin for an SML-like page DSL in WordPress.

## Features

- Custom post type: `sml_page`
- Source field (DSL) in wp-admin
- Monaco editor in wp-admin
- Menu sub-items under `SML Pages`:
  - `Templates` (global Twig template registry, Monaco editor)
  - `Markdown Files` (global `.md` parts, Monaco Markdown editor)
- Compile on save: `SML -> HTML`
- Frontend template modes for `sml_page`:
  - `Canvas` (full viewport, no theme container)
  - `Theme` (with active theme header/footer)
- Optional page-level asset includes via SML `Assets` tree with whitelist
- Supports components:
  - `Page`
  - `Hero` (Twig template: `templates/hero.twig`)
  - `Row`
  - `Column`
  - `Card` (`title`, `subtitle`)
  - `Link` (`href`, `text`, `class`, `target`)
  - `Markdown` (`text` or `part`)
  - `Image` (`src` or `str`)
  - `Spacer` (`amount`)
- No default frontend CSS/JS from plugin; uses theme assets unless SML `Assets` are defined
- Optional Twig rendering layer for semantic nodes (mapping-based)
- Uses `language-configuration.json` + `sml.tmLanguage.json` as basis for editor behavior/highlighting

## Install

1. Copy folder `sml-wp-plugin` to `wp-content/plugins/`.
2. Optional (for Twig rendering): run `composer install` inside the plugin folder.
3. Activate plugin `SML Pages MVP`.
4. Create a new `SML Page` in admin.
5. In `Template Mode`, choose `Canvas` to break out of Bootstrap/container constraints.

## Example DSL

```txt
Page {
  Assets {
    Head {
      CssTemplate { name: "bootstrap" }
      CssTemplate { name: "tailwind" }
    }
    Foot {
      JsTemplate { name: "bootstrap" }
    }
  }
  title: "Spiel des Lebens"
  padding: 8
  bgColor: "#1f2937"
  color: "#ffffff"

  Column {
    padding: 8, 16, 16, 4

    Image { src: "https://example.com/ei.png" }
    Image { src: "https://example.com/hero.png" width: 320 height: 180 }
    Spacer { amount: 32 }

    Card {
      title: "LazyRow Example"
      subtitle: "Favouriten"
      Markdown { text: "Nice **content**\\nGeht das auch?" }
    }

    Link {
      href: "https://example.com"
      text: "Mehr erfahren"
      class: "btn btn-primary"
      target: "_blank"
    }

    Markdown { part: "home.md" }
  }
}
```

Hinweis: `bgColor` und `color` funktionieren auf `Page`-Ebene (und auch auf anderen Nodes).
Du kannst zusätzlich Theme-Klassen setzen, z. B. `class: "white-row col-md-6"`.

## Twig rendering (optional)

If `twig/twig` is installed (`vendor/autoload.php` exists), the renderer first tries Twig templates for mapped semantic elements:

- `Page` -> `templates/page.twig`
- `Hero` -> `templates/hero.twig`

Context includes SML properties as direct variables (e.g. `headline`, `subheadline`) plus:

- `content` (rendered child HTML)
- `children` (raw parsed child nodes)
- `props` (all properties as array)

Twig helper functions:
- `sml_markdown_part('name.md')`
- `sml_lang()`
- `sml_css('https://.../file.css')`
- `sml_js('https://.../file.js')`

If Twig is unavailable, a mapped template is missing, or rendering fails, the plugin automatically falls back to the existing HTML renderer.

### Global templates

Use the `Templates` menu to define reusable Twig templates by name (for example `page.twig`, `hero.twig`).
These templates are available globally and are used before filesystem templates.

### Markdown parts from admin

Use the `Markdown Files` menu to create parts like `home.md`.
Then reference them in SML:

```txt
Markdown { part: "home.md" }
```

Resolution order:
1. `Markdown Files` entries in WordPress admin
2. Filesystem fallback: `wp-content/uploads/sml-parts/<name>.md`

### Page assets (whitelist)

On the root `Page` node you can define assets with `Assets -> Head/Foot`:

```txt
Page {
  Assets {
    Head {
      CssTemplate { name: "bootstrap" }
      CssTemplate { name: "tailwind" }
    }
    Foot {
      JsTemplate { name: "bootstrap" }
    }
  }
}
```

Currently available keys:
- `bootstrap` (CSS + JS bundle)
- `tailwind` (CSS)
- `pico` (CSS)

## Notes

- This is a focused MVP to get the DSL flow running.
- Monaco uses a lightweight Monarch tokenizer derived from your SML grammar file.

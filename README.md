# SML Pages MVP (WordPress Plugin)

MVP plugin for an SML-like page DSL in WordPress.

## Features

- Custom post type: `sml_page`
- Source field (DSL) in wp-admin
- Monaco editor in wp-admin
- Compile on save: `SML -> HTML`
- Frontend template modes for `sml_page`:
  - `Canvas` (full viewport, no theme container)
  - `Theme` (with active theme header/footer)
- Supports components:
  - `Page`
  - `Row`
  - `Column`
  - `Card` (`title`, `subtitle`)
  - `Link` (`href`, `text`, `class`, `target`)
  - `Markdown` (`text` or `part`)
  - `Image` (`src` or `str`)
  - `Spacer` (`amount`)
- Uses Pico.css on frontend
- Uses `language-configuration.json` + `sml.tmLanguage.json` as basis for editor behavior/highlighting

## Install

1. Copy folder `sml-wp-plugin` to `wp-content/plugins/`.
2. Activate plugin `SML Pages MVP`.
3. Create a new `SML Page` in admin.
4. In `Template Mode`, choose `Canvas` to break out of Bootstrap/container constraints.

## Example DSL

```txt
Page {
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

## Markdown parts

`Markdown { part: "home.md" }` resolves to:

`wp-content/uploads/sml-parts/home.md`

## Notes

- This is a focused MVP to get the DSL flow running.
- Monaco uses a lightweight Monarch tokenizer derived from your SML grammar file.

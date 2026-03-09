# Spec: Extend SML WordPress Plugin with Twig Rendering

## Goal

Extend the existing SML WordPress plugin so that semantic SML elements can be rendered through Twig templates instead of only being converted directly to raw HTML.

This should make structures like the following possible:

```
Page {
    Hero {
        headline: "Build native apps without a browser runtime"
        subheadline: "Forge uses SML and a native runtime instead of Electron"
    }
}
```

## Requirements

### 1. Add Twig integration

- Integrate Twig into the WordPress plugin.
- Use Twig as an optional rendering layer for SML elements.
- Each SML element should map to a Twig template.

### 2. Semantic element rendering

Implement template-based rendering for at least the following SML elements:

- Page
- Hero

Nested elements must be supported, for example:

```
Page {
    Hero {}
}
```

### 3. Template mapping

Introduce a simple mapping system:

- `Page` → `page.twig`
- `Hero` → `hero.twig`

The mapping should be easy to extend later for additional elements such as:

- Section
- FeatureCard
- BulletList
- CTA
- FAQ

### 4. Context passing

Pass SML properties into Twig templates as context variables.

Example SML:

```
Hero {
    headline: "Hello"
    subheadline: "World"
}
```

The Twig template should receive:

- `headline`
- `subheadline`

Child elements should be passed as rendered content where necessary.

### 5. Default fallback behavior

If no Twig template exists for an SML element:

- Use the current fallback renderer, or
- Render a safe placeholder instead of failing.

### 6. WordPress compatibility

- The plugin must remain fully WordPress compatible.
- Twig templates should live in a dedicated `templates/` folder.
- Rendering must work within the existing shortcode/plugin rendering flow.

### Suggested folder structure

```
templates/
    page.twig
    hero.twig
```

### Example Twig templates

**page.twig**

```
<div class="sml-page">
    {{ content|raw }}
</div>
```

**hero.twig**

```
<section class="hero">
    <h1>{{ headline }}</h1>
    <p>{{ subheadline }}</p>
</section>
```

## Expected outcome

The plugin should be able to parse and render SML like this:

```
Page {
    Hero {
        headline: "Build native apps without a browser runtime"
        subheadline: "Forge uses SML and a native runtime instead of Electron"
    }
}
```

`Page` should render its child `Hero` element using Twig templates.

## Notes

- Keep the design clean and extensible.
- Avoid a large hardcoded switch for rendering logic if possible.
- Prefer a registry or mapping-based renderer design.

<?php

if (!defined('ABSPATH')) {
    exit;
}

class SML_Renderer
{
    public function render(array $nodes): string
    {
        $out = '';
        foreach ($nodes as $node) {
            $out .= $this->renderNode($node);
        }
        return $out;
    }

    private function renderNode(array $node): string
    {
        $type = strtolower((string) ($node['type'] ?? ''));
        $props = is_array($node['props'] ?? null) ? $node['props'] : [];
        $children = is_array($node['children'] ?? null) ? $node['children'] : [];

        return match ($type) {
            'page' => $this->renderContainer('main', 'sml-page', $props, $children),
            'row' => $this->renderContainer('div', 'sml-row', $props, $children),
            'column' => $this->renderContainer('div', 'sml-column', $props, $children),
            'card' => $this->renderCard($props, $children),
            'link' => $this->renderLink($props, $children),
            'markdown' => $this->renderMarkdown($props),
            'image' => $this->renderImage($props),
            'spacer' => $this->renderSpacer($props),
            default => $this->renderContainer('div', 'sml-node sml-' . sanitize_html_class($type), $props, $children),
        };
    }

    private function renderContainer(string $tag, string $class, array $props, array $children): string
    {
        $style = $this->buildStyle($props);
        $class_attr = $this->buildClassAttr($class, $props);
        $content = '';

        foreach ($children as $child) {
            $content .= $this->renderNode($child);
        }

        return '<' . $tag . ' class="' . esc_attr($class_attr) . '"' . $style . '>' . $content . '</' . $tag . '>';
    }

    private function renderImage(array $props): string
    {
        $src = (string) ($props['src'] ?? $props['str'] ?? '');
        if ($src === '') {
            return '';
        }

        $alt = (string) ($props['alt'] ?? '');
        $style = $this->buildStyle($props);
        $dimension_style = $this->buildImageDimensionStyle($props);
        if ($dimension_style !== '') {
            $style = $this->appendStyle($style, $dimension_style);
        }

        $class_attr = $this->buildClassAttr('sml-image', $props);
        return '<img class="' . esc_attr($class_attr) . '" src="' . esc_url($src) . '" alt="' . esc_attr($alt) . '"' . $style . ' />';
    }

    private function renderSpacer(array $props): string
    {
        $amount = $props['amount'] ?? 16;
        $px = is_numeric($amount) ? (float) $amount : 16;
        return '<div class="sml-spacer" style="height:' . esc_attr((string) $px) . 'px"></div>';
    }

    private function renderMarkdown(array $props): string
    {
        $markdown = '';

        if (isset($props['text'])) {
            $markdown = (string) $props['text'];
        } elseif (isset($props['part'])) {
            $markdown = $this->loadPart((string) $props['part']);
        }

        $html = $this->markdownToHtml($markdown);
        $style = $this->buildStyle($props);

        $class_attr = $this->buildClassAttr('sml-markdown', $props);
        return '<div class="' . esc_attr($class_attr) . '"' . $style . '>' . $html . '</div>';
    }

    private function renderCard(array $props, array $children): string
    {
        $style = $this->buildStyle($props);
        $title = isset($props['title']) ? (string) $props['title'] : '';
        $subtitle = isset($props['subtitle']) ? (string) $props['subtitle'] : '';

        $inner = '';
        if ($title !== '') {
            $inner .= '<h3 class="sml-card-title">' . esc_html($title) . '</h3>';
        }
        if ($subtitle !== '') {
            $inner .= '<p class="sml-card-subtitle">' . esc_html($subtitle) . '</p>';
        }
        foreach ($children as $child) {
            $inner .= $this->renderNode($child);
        }

        $class_attr = $this->buildClassAttr('sml-card', $props);
        return '<article class="' . esc_attr($class_attr) . '"' . $style . '><div class="sml-card-body">' . $inner . '</div></article>';
    }

    private function renderLink(array $props, array $children): string
    {
        $href = isset($props['href']) ? (string) $props['href'] : '';
        if ($href === '') {
            return '';
        }

        $text = isset($props['text']) ? (string) $props['text'] : '';
        $class_attr = $this->buildClassAttr('sml-link', $props);
        $style = $this->buildStyle($props);

        $target = isset($props['target']) ? trim((string) $props['target']) : '';
        $target_attr = '';
        $rel_attr = '';
        if ($target !== '') {
            $target_attr = ' target="' . esc_attr($target) . '"';
            if ($target === '_blank') {
                $rel_attr = ' rel="noopener noreferrer"';
            }
        }

        $content = '';
        if ($text !== '') {
            $content = esc_html($text);
        } else {
            foreach ($children as $child) {
                $content .= $this->renderNode($child);
            }
            if ($content === '') {
                $content = esc_html($href);
            }
        }

        return '<a class="' . esc_attr($class_attr) . '" href="' . esc_url($href) . '"' . $target_attr . $rel_attr . $style . '>' . $content . '</a>';
    }

    private function loadPart(string $part): string
    {
        $part = ltrim($part, '/');
        $part = str_replace('..', '', $part);

        $upload = wp_upload_dir();
        $base = trailingslashit($upload['basedir']) . 'sml-parts/';
        $path = $base . $part;

        if (!is_file($path) || !is_readable($path)) {
            return 'Part not found: ' . $part;
        }

        $content = file_get_contents($path);
        return $content === false ? '' : $content;
    }

    private function markdownToHtml(string $markdown): string
    {
        $lines = preg_split('/\R/', $markdown) ?: [];
        $html = '';
        $paragraph = [];

        $flushParagraph = static function () use (&$paragraph, &$html): void {
            if (empty($paragraph)) {
                return;
            }
            $text = implode("\n", $paragraph);
            $text = wp_kses_post($text);
            $text = preg_replace('/\*\*(.*?)\*\*/', '<strong>$1</strong>', $text);
            $text = preg_replace('/\*(.*?)\*/', '<em>$1</em>', $text);
            // Keep author-intended line breaks inside one paragraph.
            $text = preg_replace("/\r\n|\r|\n/", "<br />\n", $text);
            $html .= '<p>' . $text . '</p>';
            $paragraph = [];
        };

        foreach ($lines as $line) {
            $trimmed = trim($line);

            if ($trimmed === '') {
                $flushParagraph();
                continue;
            }

            if (preg_match('/^(#{1,6})\s+(.*)$/', $trimmed, $matches)) {
                $flushParagraph();
                $level = strlen($matches[1]);
                $text = wp_kses_post($matches[2]);
                $html .= '<h' . $level . '>' . $text . '</h' . $level . '>';
                continue;
            }

            if (preg_match('/^[-*]\s+(.*)$/', $trimmed, $matches)) {
                $flushParagraph();
                $text = wp_kses_post($matches[1]);
                $html .= '<ul><li>' . $text . '</li></ul>';
                continue;
            }

            $paragraph[] = $trimmed;
        }

        $flushParagraph();

        return $html;
    }

    private function buildStyle(array $props): string
    {
        $styles = [];

        if (isset($props['padding'])) {
            $styles[] = 'padding:' . $this->spacingValue($props['padding']);
        }

        if (isset($props['bgColor'])) {
            $bg = $this->sanitizeCssColor((string) $props['bgColor']);
            if ($bg !== '') {
                $styles[] = 'background-color:' . $bg;
            }
        }

        if (isset($props['color'])) {
            $color = $this->sanitizeCssColor((string) $props['color']);
            if ($color !== '') {
                $styles[] = 'color:' . $color;
            }
        }

        if (isset($props['gap'])) {
            $styles[] = 'gap:' . $this->numericUnit($props['gap']) . ';display:flex;flex-direction:column';
        }

        if (isset($props['scrollable']) && $props['scrollable'] === true) {
            $styles[] = 'overflow:auto';
        }

        return empty($styles) ? '' : ' style="' . esc_attr(implode(';', $styles)) . '"';
    }

    private function spacingValue(mixed $value): string
    {
        if (is_array($value)) {
            $parts = array_map(fn($v) => $this->numericUnit($v), $value);
            return implode(' ', array_slice($parts, 0, 4));
        }

        return $this->numericUnit($value);
    }

    private function numericUnit(mixed $value): string
    {
        if (is_numeric($value)) {
            return (string) $value . 'px';
        }

        $string = trim((string) $value);
        if ($string === '') {
            return '0';
        }

        if (preg_match('/^-?\d+(\.\d+)?(px|rem|em|%|vh|vw)$/', $string)) {
            return $string;
        }

        if (preg_match('/^-?\d+(\.\d+)?$/', $string)) {
            return $string . 'px';
        }

        return '0';
    }

    private function sanitizeCssColor(string $value): string
    {
        $value = trim($value);
        if ($value === '') {
            return '';
        }

        // Hex colors: #RGB, #RRGGBB, #RGBA, #RRGGBBAA
        if (preg_match('/^#(?:[A-Fa-f0-9]{3}|[A-Fa-f0-9]{4}|[A-Fa-f0-9]{6}|[A-Fa-f0-9]{8})$/', $value)) {
            return $value;
        }

        // rgb()/rgba()/hsl()/hsla()
        if (preg_match('/^(?:rgb|rgba|hsl|hsla)\\([^\\)]+\\)$/i', $value)) {
            return $value;
        }

        // Named colors (basic safety)
        if (preg_match('/^[a-zA-Z]+$/', $value)) {
            return strtolower($value);
        }

        return '';
    }

    private function buildClassAttr(string $baseClass, array $props): string
    {
        $classes = [$baseClass];

        foreach (['class', 'classes'] as $key) {
            if (!isset($props[$key])) {
                continue;
            }

            $raw = $props[$key];
            if (is_array($raw)) {
                $raw = implode(' ', array_map(static fn($v) => (string) $v, $raw));
            }

            $raw = trim((string) $raw);
            if ($raw === '') {
                continue;
            }

            foreach (preg_split('/\s+/', $raw) as $cls) {
                $cls = sanitize_html_class($cls);
                if ($cls !== '') {
                    $classes[] = $cls;
                }
            }
        }

        $classes = array_values(array_unique($classes));
        return implode(' ', $classes);
    }

    private function buildImageDimensionStyle(array $props): string
    {
        $styles = [];

        if (isset($props['width'])) {
            $w = $this->numericUnit($props['width']);
            if ($w !== '0') {
                $styles[] = 'width:' . $w;
            }
        }

        if (isset($props['height'])) {
            $h = $this->numericUnit($props['height']);
            if ($h !== '0') {
                $styles[] = 'height:' . $h;
            }
        }

        return implode(';', $styles);
    }

    private function appendStyle(string $existingStyleAttr, string $extraDeclarations): string
    {
        if ($extraDeclarations === '') {
            return $existingStyleAttr;
        }

        if ($existingStyleAttr === '') {
            return ' style="' . esc_attr($extraDeclarations) . '"';
        }

        $existing = preg_replace('/^ style="/', '', $existingStyleAttr);
        $existing = preg_replace('/"$/', '', (string) $existing);
        $merged = rtrim((string) $existing, ';');

        if ($merged !== '') {
            $merged .= ';';
        }
        $merged .= $extraDeclarations;

        return ' style="' . esc_attr($merged) . '"';
    }
}

<?php

if (!defined('ABSPATH')) {
    exit;
}

class SML_Renderer
{
    private ?object $twig = null;
    private ?object $twig_loader = null;
    /**
     * @var array<string, bool>
     */
    private array $emitted_css = [];
    /**
     * @var array<string, bool>
     */
    private array $emitted_js = [];

    /**
     * @var array<string, string>
     */
    private array $template_map = [
        'page' => 'page.twig',
        'hero' => 'hero.twig',
    ];

    /**
     * @var array<string, string>
     */
    private array $template_overrides = [];

    public function __construct(?string $template_dir = null, ?array $template_map = null, ?array $template_overrides = null)
    {
        if (is_array($template_map)) {
            $this->setTemplateMap($template_map);
        }
        if (is_array($template_overrides)) {
            $this->setTemplateOverrides($template_overrides);
        }

        if ($template_dir === null) {
            $template_dir = dirname(__DIR__) . '/templates';
        }

        $this->initTwig($template_dir);
    }

    /**
     * @param array<string, string> $map
     */
    public function setTemplateMap(array $map): void
    {
        $normalized = [];
        foreach ($map as $node => $template) {
            $node = strtolower(trim((string) $node));
            $template = trim((string) $template);
            if ($node === '' || $template === '') {
                continue;
            }
            $normalized[$node] = $template;
        }

        if ($normalized !== []) {
            $this->template_map = array_merge($this->template_map, $normalized);
        }
    }

    public function registerTemplate(string $node, string $template): void
    {
        $node = strtolower(trim($node));
        $template = trim($template);
        if ($node === '' || $template === '') {
            return;
        }

        $this->template_map[$node] = $template;
    }

    /**
     * @param array<string, string> $overrides keyed by template name (e.g. page.twig)
     */
    public function setTemplateOverrides(array $overrides): void
    {
        $normalized = [];
        foreach ($overrides as $name => $content) {
            $name = trim((string) $name);
            if ($name === '' || !str_ends_with($name, '.twig')) {
                continue;
            }
            $normalized[$name] = (string) $content;
        }
        $this->template_overrides = $normalized;
    }

    public function render(array $nodes): string
    {
        $out = '';
        foreach ($nodes as $node) {
            $out .= $this->renderNode($node);
        }
        return $out;
    }

    private function initTwig(string $template_dir): void
    {
        if (!class_exists('Twig\\Loader\\FilesystemLoader') || !class_exists('Twig\\Environment')) {
            return;
        }

        if (!is_dir($template_dir)) {
            return;
        }

        try {
            $filesystem_loader = new \Twig\Loader\FilesystemLoader($template_dir);
            $this->twig_loader = $filesystem_loader;

            if ($this->template_overrides !== [] && class_exists('Twig\\Loader\\ArrayLoader') && class_exists('Twig\\Loader\\ChainLoader')) {
                $array_loader = new \Twig\Loader\ArrayLoader($this->template_overrides);
                $this->twig_loader = new \Twig\Loader\ChainLoader([$array_loader, $filesystem_loader]);
            }

            $this->twig = new \Twig\Environment($this->twig_loader, [
                'cache' => false,
                'autoescape' => 'html',
                'strict_variables' => false,
            ]);

            if (class_exists('Twig\\TwigFunction')) {
                $this->twig->addFunction(new \Twig\TwigFunction('sml_markdown_part', function (string $part): string {
                    $raw = $this->loadPart($part);
                    return $this->markdownToHtml($raw);
                }, ['is_safe' => ['html']]));

                $this->twig->addFunction(new \Twig\TwigFunction('sml_lang', function (): string {
                    return $this->getCurrentLanguage();
                }));

                $this->twig->addFunction(new \Twig\TwigFunction('sml_css', function (string $url): string {
                    return $this->renderExternalAssetTag($url, 'css');
                }, ['is_safe' => ['html']]));

                $this->twig->addFunction(new \Twig\TwigFunction('sml_js', function (string $url): string {
                    return $this->renderExternalAssetTag($url, 'js');
                }, ['is_safe' => ['html']]));
            }
        } catch (Throwable) {
            $this->twig_loader = null;
            $this->twig = null;
        }
    }

    private function renderNode(array $node): string
    {
        $type = strtolower((string) ($node['type'] ?? ''));
        $props = is_array($node['props'] ?? null) ? $node['props'] : [];
        $children = is_array($node['children'] ?? null) ? $node['children'] : [];

        $twig_output = $this->renderNodeWithTwig($type, $props, $children);
        if ($twig_output !== null) {
            return $twig_output;
        }

        return $this->renderNodeFallback($type, $props, $children);
    }

    private function renderNodeWithTwig(string $type, array $props, array $children): ?string
    {
        $template = $this->template_map[$type] ?? '';
        if ($template === '' || $this->twig === null || $this->twig_loader === null) {
            return null;
        }

        if (!$this->twigTemplateExists($template)) {
            return null;
        }

        $content = $this->renderChildren($children);
        $context = $props;
        $class_attr = $this->buildClassAttr($this->defaultClassForType($type), $props);
        $style_attr = $this->buildStyle($props);
        $context['props'] = $props;
        $context['children'] = $children;
        $context['content'] = $content;
        $context['type'] = $type;
        $context['class_attr'] = $class_attr;
        $context['style_attr'] = $style_attr;
        $context['attrs'] = ' class="' . esc_attr($class_attr) . '"' . $style_attr;
        $context['lang'] = $this->getCurrentLanguage();

        try {
            $html = $this->twig->render($template, $context);
            return is_string($html) ? $html : null;
        } catch (Throwable) {
            return null;
        }
    }

    private function twigTemplateExists(string $template): bool
    {
        if ($this->twig_loader === null || !method_exists($this->twig_loader, 'exists')) {
            return false;
        }

        try {
            return (bool) $this->twig_loader->exists($template);
        } catch (Throwable) {
            return false;
        }
    }

    private function renderNodeFallback(string $type, array $props, array $children): string
    {
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

    private function renderChildren(array $children): string
    {
        $content = '';
        foreach ($children as $child) {
            $content .= $this->renderNode($child);
        }

        return $content;
    }

    private function defaultClassForType(string $type): string
    {
        return match ($type) {
            'page' => 'sml-page',
            'hero' => 'sml-hero',
            default => 'sml-node sml-' . sanitize_html_class($type),
        };
    }

    private function renderContainer(string $tag, string $class, array $props, array $children): string
    {
        $style = $this->buildStyle($props);
        $class_attr = $this->buildClassAttr($class, $props);
        $content = $this->renderChildren($children);

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
        $inner .= $this->renderChildren($children);

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
            $content = $this->renderChildren($children);
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

        if (function_exists('sml_pages_get_markdown_part_content')) {
            $db_part = sml_pages_get_markdown_part_content($part);
            if (is_string($db_part) && $db_part !== '') {
                return $db_part;
            }
        }

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

    private function getCurrentLanguage(): string
    {
        // Polylang
        if (function_exists('pll_current_language')) {
            $lang = pll_current_language('slug');
            if (is_string($lang) && $lang !== '') {
                return strtolower($lang);
            }
        }

        // WPML
        if (function_exists('apply_filters')) {
            $lang = apply_filters('wpml_current_language', null);
            if (is_string($lang) && $lang !== '') {
                return strtolower($lang);
            }
        }

        if (function_exists('determine_locale')) {
            $locale = (string) determine_locale();
            if ($locale !== '') {
                return strtolower(substr($locale, 0, 2));
            }
        }

        $locale = (string) get_locale();
        if ($locale !== '') {
            return strtolower(substr($locale, 0, 2));
        }

        return 'en';
    }

    private function renderExternalAssetTag(string $url, string $type): string
    {
        $safe_url = $this->sanitizeExternalAssetUrl($url);
        if ($safe_url === '') {
            return '';
        }

        if ($type === 'css') {
            if (isset($this->emitted_css[$safe_url])) {
                return '';
            }
            $this->emitted_css[$safe_url] = true;
            return '<link rel="stylesheet" href="' . esc_url($safe_url) . '" />';
        }

        if ($type === 'js') {
            if (isset($this->emitted_js[$safe_url])) {
                return '';
            }
            $this->emitted_js[$safe_url] = true;
            return '<script src="' . esc_url($safe_url) . '" defer></script>';
        }

        return '';
    }

    private function sanitizeExternalAssetUrl(string $url): string
    {
        $url = trim($url);
        if ($url === '') {
            return '';
        }

        $validated = esc_url_raw($url, ['https']);
        return is_string($validated) ? $validated : '';
    }
}

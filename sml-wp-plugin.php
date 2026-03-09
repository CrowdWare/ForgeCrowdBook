<?php
/**
 * Plugin Name: Forge WP SML Compiler
 * Plugin URI: https://codeberg.org/CrowdWare/forge-wp-sml-compiler
 * Description: SML Compiler for WordPress: build with SML/Twig/Markdown and ship super fast static HTML output.
 * Version: 0.1.12
 * Author: Artanidos
 * Author URI: https://codeberg.org/CrowdWare
 */

if (!defined('ABSPATH')) {
    exit;
}

$autoload = __DIR__ . '/vendor/autoload.php';
if (is_readable($autoload)) {
    require_once $autoload;
}

require_once __DIR__ . '/includes/class-sml-parser.php';
require_once __DIR__ . '/includes/class-sml-renderer.php';

class SML_Pages_Plugin
{
    public const META_SOURCE = '_sml_source';
    public const META_RENDERED = '_sml_rendered_html';
    public const META_TEMPLATE_MODE = '_sml_template_mode';
    public const META_TEMPLATE_NAME = '_sml_template_name';
    public const META_TEMPLATE_SOURCE = '_sml_template_source';
    public const META_MD_PART_NAME = '_sml_md_part_name';
    public const META_MD_PART_SOURCE = '_sml_md_part_source';
    public const META_PAGE_ASSETS = '_sml_page_assets';

    /**
     * @var array<string, array{css?: string, js?: string, deps?: array<int, string>}>
     */
    private const ASSET_REGISTRY = [
        'bootstrap' => [
            'css' => 'https://cdn.jsdelivr.net/npm/bootstrap@5.3.7/dist/css/bootstrap.min.css',
            'js' => 'https://cdn.jsdelivr.net/npm/bootstrap@5.3.7/dist/js/bootstrap.bundle.min.js',
        ],
        'tailwind' => [
            'css' => 'https://cdn.jsdelivr.net/npm/tailwindcss@2.2.19/dist/tailwind.min.css',
        ],
        'pico' => [
            'css' => 'https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css',
        ],
    ];

    public function __construct()
    {
        add_action('init', [$this, 'register_post_type']);
        add_action('add_meta_boxes', [$this, 'add_meta_boxes']);
        add_action('save_post_sml_page', [$this, 'save_sml_page'], 10, 2);
        add_action('save_post_sml_template', [$this, 'save_sml_template'], 10, 2);
        add_action('save_post_sml_markdown_part', [$this, 'save_sml_markdown_part'], 10, 2);
        add_filter('template_include', [$this, 'template_include']);
        add_shortcode('sml_page', [$this, 'shortcode_sml_page']);

        add_action('admin_enqueue_scripts', [$this, 'admin_assets']);
        add_action('wp_enqueue_scripts', [$this, 'frontend_assets']);
    }

    public function register_post_type(): void
    {
        register_post_type('sml_page', [
            'label' => 'SML Pages',
            'public' => true,
            'publicly_queryable' => true,
            'show_in_rest' => true,
            'menu_icon' => 'dashicons-editor-code',
            'supports' => ['title', 'excerpt'],
            'has_archive' => true,
            'rewrite' => ['slug' => 'sml', 'with_front' => false],
        ]);

        register_post_type('sml_template', [
            'label' => 'Templates',
            'public' => false,
            'show_ui' => true,
            'show_in_menu' => 'edit.php?post_type=sml_page',
            'show_in_rest' => false,
            'supports' => ['title'],
            'menu_icon' => 'dashicons-media-code',
            'capability_type' => 'post',
        ]);

        register_post_type('sml_markdown_part', [
            'label' => 'Markdown Files',
            'public' => false,
            'show_ui' => true,
            'show_in_menu' => 'edit.php?post_type=sml_page',
            'show_in_rest' => false,
            'supports' => ['title'],
            'menu_icon' => 'dashicons-media-text',
            'capability_type' => 'post',
        ]);
    }

    public function add_meta_boxes(): void
    {
        add_meta_box(
            'sml_source_editor',
            'SML Source',
            [$this, 'render_source_metabox'],
            'sml_page',
            'normal',
            'high'
        );

        add_meta_box(
            'sml_render_preview',
            'Rendered Preview (cached)',
            [$this, 'render_preview_metabox'],
            'sml_page',
            'normal',
            'default'
        );

        add_meta_box(
            'sml_template_editor',
            'Template Source',
            [$this, 'render_template_source_metabox'],
            'sml_template',
            'normal',
            'high'
        );

        add_meta_box(
            'sml_markdown_editor',
            'Markdown Source',
            [$this, 'render_markdown_part_metabox'],
            'sml_markdown_part',
            'normal',
            'high'
        );
    }

    public function render_source_metabox(WP_Post $post): void
    {
        wp_nonce_field('sml_save_source', 'sml_source_nonce');
        $source = (string) get_post_meta($post->ID, self::META_SOURCE, true);
        $template_mode = (string) get_post_meta($post->ID, self::META_TEMPLATE_MODE, true);
        if (!in_array($template_mode, ['theme', 'canvas'], true)) {
            $template_mode = 'canvas';
        }

        if ($source === '') {
            $source = "Page {\n  padding: 16\n  Column {\n    padding: 8\n    Markdown { text: \"# Hello SML\" }\n    Spacer { amount: 16 }\n    Markdown { text: \"Build once, render anywhere.\" }\n  }\n}";
        }

        echo '<p>Use your SML DSL here. Supported: Page, Hero, Row, Column, Card, Link, Markdown, Image, Spacer.</p>';
        echo '<p><label for="sml_template_mode"><strong>Template Mode:</strong></label> ';
        echo '<select id="sml_template_mode" name="sml_template_mode">';
        echo '<option value="canvas"' . selected($template_mode, 'canvas', false) . '>Canvas (full viewport)</option>';
        echo '<option value="theme"' . selected($template_mode, 'theme', false) . '>Theme (with header/footer)</option>';
        echo '</select></p>';
        echo '<p><small>Assets via SML: <code>Assets { Head { CssTemplate { name: "bootstrap" } CssTemplate { name: "pico" } } Foot { JsTemplate { name: "bootstrap" } } }</code>.</small></p>';
        echo '<div id="sml_monaco_editor" aria-label="SML Monaco Editor"></div>';
        echo '<textarea id="sml_source" name="sml_source" style="width:100%;min-height:380px;font-family:monospace;">' . esc_textarea($source) . '</textarea>';
        echo '<p><small>Markdown supports <code>text</code> or <code>part</code>. Parts resolve from the <strong>Markdown Files</strong> menu (fallback: <code>wp-content/uploads/sml-parts/</code>).</small></p>';
    }

    public function render_preview_metabox(WP_Post $post): void
    {
        $html = (string) get_post_meta($post->ID, self::META_RENDERED, true);
        if ($html === '') {
            echo '<p>No cached render yet. Save this post to compile.</p>';
            return;
        }

        echo '<div style="border:1px solid #ddd;padding:12px;max-height:320px;overflow:auto;background:#fff;">' . wp_kses_post($html) . '</div>';
    }

    public function render_template_source_metabox(WP_Post $post): void
    {
        wp_nonce_field('sml_save_template', 'sml_template_nonce');

        $name = (string) get_post_meta($post->ID, self::META_TEMPLATE_NAME, true);
        $source = (string) get_post_meta($post->ID, self::META_TEMPLATE_SOURCE, true);

        if ($name === '') {
            $name = sanitize_title($post->post_title);
            if ($name === '') {
                $name = 'template';
            }
            $name .= '.twig';
        }

        echo '<p><label for="sml_template_name"><strong>Template Name</strong> (e.g. <code>page.twig</code>, <code>hero.twig</code>)</label></p>';
        echo '<input type="text" id="sml_template_name" name="sml_template_name" value="' . esc_attr($name) . '" style="width:100%;max-width:420px;" />';
        echo '<p><small>This name is used by the renderer mapping (Page -> page.twig, Hero -> hero.twig, ...).</small></p>';

        echo '<div id="sml_template_monaco_editor" class="sml-twig-editor" aria-label="Twig Template Monaco Editor"></div>';
        echo '<textarea id="sml_template_source" name="sml_template_source" style="width:100%;min-height:320px;font-family:monospace;">' . esc_textarea($source) . '</textarea>';
    }

    public function render_markdown_part_metabox(WP_Post $post): void
    {
        wp_nonce_field('sml_save_markdown_part', 'sml_markdown_part_nonce');

        $name = (string) get_post_meta($post->ID, self::META_MD_PART_NAME, true);
        $source = (string) get_post_meta($post->ID, self::META_MD_PART_SOURCE, true);

        if ($name === '') {
            $name = sanitize_title($post->post_title);
            if ($name === '') {
                $name = 'part';
            }
            $name .= '.md';
        }

        echo '<p><label for="sml_markdown_part_name"><strong>Markdown File Name</strong> (e.g. <code>home.md</code>)</label></p>';
        echo '<input type="text" id="sml_markdown_part_name" name="sml_markdown_part_name" value="' . esc_attr($name) . '" style="width:100%;max-width:420px;" />';
        echo '<p><small>Use this with <code>Markdown { part: "' . esc_html($name) . '" }</code>.</small></p>';

        echo '<div id="sml_markdown_monaco_editor" class="sml-markdown-editor" aria-label="Markdown Monaco Editor"></div>';
        echo '<textarea id="sml_markdown_source" name="sml_markdown_source" style="width:100%;min-height:320px;font-family:monospace;">' . esc_textarea($source) . '</textarea>';
    }

    public function save_sml_page(int $post_id, WP_Post $post): void
    {
        if (!isset($_POST['sml_source_nonce']) || !wp_verify_nonce(sanitize_text_field(wp_unslash($_POST['sml_source_nonce'])), 'sml_save_source')) {
            return;
        }

        if (!current_user_can('edit_post', $post_id)) {
            return;
        }

        $source = isset($_POST['sml_source']) ? wp_unslash($_POST['sml_source']) : '';
        if (!is_string($source)) {
            $source = '';
        }
        $template_mode = isset($_POST['sml_template_mode']) ? sanitize_text_field(wp_unslash($_POST['sml_template_mode'])) : 'canvas';
        if (!in_array($template_mode, ['theme', 'canvas'], true)) {
            $template_mode = 'canvas';
        }

        update_post_meta($post_id, self::META_SOURCE, $source);
        update_post_meta($post_id, self::META_TEMPLATE_MODE, $template_mode);
        $rendered = $this->compile_source($source);
        update_post_meta($post_id, self::META_RENDERED, $rendered);

        $assets = $this->extract_page_assets_from_source($source);
        if ($assets['css'] === [] && $assets['js'] === []) {
            delete_post_meta($post_id, self::META_PAGE_ASSETS);
        } else {
            update_post_meta($post_id, self::META_PAGE_ASSETS, wp_json_encode($assets));
        }
    }

    public function save_sml_template(int $post_id, WP_Post $post): void
    {
        if (!isset($_POST['sml_template_nonce']) || !wp_verify_nonce(sanitize_text_field(wp_unslash($_POST['sml_template_nonce'])), 'sml_save_template')) {
            return;
        }
        if (!current_user_can('edit_post', $post_id)) {
            return;
        }

        $name = isset($_POST['sml_template_name']) ? sanitize_text_field(wp_unslash($_POST['sml_template_name'])) : '';
        $source = isset($_POST['sml_template_source']) ? wp_unslash($_POST['sml_template_source']) : '';
        if (!is_string($source)) {
            $source = '';
        }
        $name = self::sanitize_template_filename($name);
        if ($name === '') {
            $name = 'template-' . $post_id . '.twig';
        }

        update_post_meta($post_id, self::META_TEMPLATE_NAME, $name);
        update_post_meta($post_id, self::META_TEMPLATE_SOURCE, $source);

        $this->recompile_pages_for_template($name);
    }

    public function save_sml_markdown_part(int $post_id, WP_Post $post): void
    {
        if (!isset($_POST['sml_markdown_part_nonce']) || !wp_verify_nonce(sanitize_text_field(wp_unslash($_POST['sml_markdown_part_nonce'])), 'sml_save_markdown_part')) {
            return;
        }
        if (!current_user_can('edit_post', $post_id)) {
            return;
        }

        $name = isset($_POST['sml_markdown_part_name']) ? sanitize_text_field(wp_unslash($_POST['sml_markdown_part_name'])) : '';
        $source = isset($_POST['sml_markdown_source']) ? wp_unslash($_POST['sml_markdown_source']) : '';
        if (!is_string($source)) {
            $source = '';
        }
        $name = self::sanitize_markdown_filename($name);
        if ($name === '') {
            $name = 'part-' . $post_id . '.md';
        }

        update_post_meta($post_id, self::META_MD_PART_NAME, $name);
        update_post_meta($post_id, self::META_MD_PART_SOURCE, $source);

        $this->recompile_pages_for_markdown_part($name);
    }

    private function compile_source(string $source): string
    {
        try {
            $parser = new SML_Parser();
            $nodes = $parser->parse($source);

            $global_templates = self::get_global_template_overrides();

            $renderer = new SML_Renderer(__DIR__ . '/templates', null, $global_templates);
            return $renderer->render($nodes);
        } catch (Throwable $e) {
            return '<pre class="sml-error">Compile error: ' . esc_html($e->getMessage()) . '</pre>';
        }
    }

    private function recompile_pages_for_markdown_part(string $part_name): void
    {
        // Markdown parts can be referenced indirectly inside Twig templates
        // (e.g. sml_markdown_part(headline ~ ".md")), so affected pages
        // cannot be determined reliably from SML source alone.
        $this->recompile_matching_pages(static function (): bool {
            return true;
        });
    }

    private function recompile_pages_for_template(string $template_name): void
    {
        $template_name = strtolower($template_name);

        if ($template_name === 'page.twig') {
            $this->recompile_matching_pages(static function (string $source): bool {
                return str_contains($source, 'Page {');
            });
            return;
        }

        if ($template_name === 'hero.twig') {
            $this->recompile_matching_pages(static function (string $source): bool {
                return str_contains($source, 'Hero {');
            });
            return;
        }

        // Unknown template mapping: safest fallback is to recompile all SML pages.
        $this->recompile_matching_pages(static function (): bool {
            return true;
        });
    }

    /**
     * @param callable(string): bool $matches
     */
    private function recompile_matching_pages(callable $matches): void
    {
        $pages = get_posts([
            'post_type' => 'sml_page',
            'post_status' => ['publish', 'draft', 'private', 'future', 'pending'],
            'numberposts' => -1,
            'fields' => 'ids',
        ]);

        if (!is_array($pages) || $pages === []) {
            return;
        }

        foreach ($pages as $page_id) {
            $page_id = (int) $page_id;
            if ($page_id <= 0) {
                continue;
            }

            $source = (string) get_post_meta($page_id, self::META_SOURCE, true);
            if ($source === '' || !$matches($source)) {
                continue;
            }

            $rendered = $this->compile_source($source);
            update_post_meta($page_id, self::META_RENDERED, $rendered);

            $assets = $this->extract_page_assets_from_source($source);
            if ($assets['css'] === [] && $assets['js'] === []) {
                delete_post_meta($page_id, self::META_PAGE_ASSETS);
            } else {
                update_post_meta($page_id, self::META_PAGE_ASSETS, wp_json_encode($assets));
            }
        }
    }

    public function template_include(string $template): string
    {
        if (is_singular('sml_page')) {
            $post_id = (int) get_queried_object_id();
            $mode = (string) get_post_meta($post_id, self::META_TEMPLATE_MODE, true);
            if (!in_array($mode, ['theme', 'canvas'], true)) {
                $mode = 'canvas';
            }

            $custom = ($mode === 'theme')
                ? __DIR__ . '/templates/single-sml_page.php'
                : __DIR__ . '/templates/single-sml_page-canvas.php';
            if (is_file($custom)) {
                return $custom;
            }
        }

        return $template;
    }

    public function shortcode_sml_page(array $atts): string
    {
        $atts = shortcode_atts(['id' => 0], $atts, 'sml_page');
        $id = (int) $atts['id'];
        if ($id <= 0) {
            return '';
        }

        $html = (string) get_post_meta($id, self::META_RENDERED, true);
        return '<div class="sml-shortcode">' . wp_kses_post($html) . '</div>';
    }

    public function admin_assets(string $hook): void
    {
        if ($hook !== 'post.php' && $hook !== 'post-new.php') {
            return;
        }

        $screen = get_current_screen();
        if (!$screen || !in_array($screen->post_type, ['sml_page', 'sml_template', 'sml_markdown_part'], true)) {
            return;
        }

        wp_enqueue_style('sml-admin', plugins_url('assets/sml-admin.css', __FILE__), [], '0.1.12');
        wp_enqueue_script('sml-monaco-loader', 'https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.52.2/min/vs/loader.min.js', [], null, true);
        wp_enqueue_script('sml-admin', plugins_url('assets/sml-admin.js', __FILE__), ['sml-monaco-loader'], '0.1.12', true);

        $language_config_path = __DIR__ . '/language-configuration.json';
        $grammar_path = __DIR__ . '/sml.tmLanguage.json';

        $language_config = [];
        if (is_readable($language_config_path)) {
            $decoded = json_decode((string) file_get_contents($language_config_path), true);
            if (is_array($decoded)) {
                $language_config = $decoded;
            }
        }

        $grammar = [];
        if (is_readable($grammar_path)) {
            $decoded = json_decode((string) file_get_contents($grammar_path), true);
            if (is_array($decoded)) {
                $grammar = $decoded;
            }
        }

        $config = [
            'vsPath' => 'https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.52.2/min/vs',
            'languageId' => 'sml',
            'languageConfiguration' => $language_config,
            'tmGrammar' => $grammar,
        ];
        wp_add_inline_script('sml-admin', 'window.SML_EDITOR_CONFIG = ' . wp_json_encode($config) . ';', 'before');
    }

    public function frontend_assets(): void
    {
        if (!is_singular('sml_page')) {
            return;
        }

        $post_id = (int) get_queried_object_id();
        if ($post_id <= 0) {
            return;
        }

        $raw = (string) get_post_meta($post_id, self::META_PAGE_ASSETS, true);
        if ($raw === '') {
            return;
        }

        $assets = json_decode($raw, true);
        if (!is_array($assets)) {
            return;
        }

        $css = is_array($assets['css'] ?? null) ? $assets['css'] : [];
        foreach ($css as $key) {
            $this->enqueue_registered_asset((string) $key, 'css');
        }

        $js = is_array($assets['js'] ?? null) ? $assets['js'] : [];
        foreach ($js as $key) {
            $this->enqueue_registered_asset((string) $key, 'js');
        }
    }

    public static function get_rendered_for_post(int $post_id): string
    {
        return (string) get_post_meta($post_id, self::META_RENDERED, true);
    }

    /**
     * @return array<string, string>
     */
    public static function get_global_template_overrides(): array
    {
        $items = get_posts([
            'post_type' => 'sml_template',
            'post_status' => ['publish', 'draft', 'private'],
            'numberposts' => -1,
        ]);

        $out = [];
        foreach ($items as $item) {
            $name = (string) get_post_meta($item->ID, self::META_TEMPLATE_NAME, true);
            $source = (string) get_post_meta($item->ID, self::META_TEMPLATE_SOURCE, true);
            $name = self::sanitize_template_filename($name);
            if ($name === '' || trim($source) === '') {
                continue;
            }
            $out[$name] = $source;
        }

        return $out;
    }

    public static function get_markdown_part_content(string $part): ?string
    {
        $name = self::sanitize_markdown_filename($part);
        if ($name === '') {
            return null;
        }

        $items = get_posts([
            'post_type' => 'sml_markdown_part',
            'post_status' => ['publish', 'draft', 'private'],
            'numberposts' => 1,
            'meta_key' => self::META_MD_PART_NAME,
            'meta_value' => $name,
        ]);

        if (!$items) {
            return null;
        }

        $source = (string) get_post_meta($items[0]->ID, self::META_MD_PART_SOURCE, true);
        return $source === '' ? null : $source;
    }

    /**
     * @return array{css: array<int, string>, js: array<int, string>}
     */
    private function extract_page_assets_from_source(string $source): array
    {
        $result = ['css' => [], 'js' => []];

        try {
            $parser = new SML_Parser();
            $nodes = $parser->parse($source);
        } catch (Throwable) {
            return $result;
        }

        $first = $nodes[0] ?? null;
        if (!is_array($first) || strtolower((string) ($first['type'] ?? '')) !== 'page') {
            return $result;
        }

        $children = is_array($first['children'] ?? null) ? $first['children'] : [];

        $from_tree = $this->extract_assets_from_page_children($children);
        $result['css'] = $from_tree['css'];
        $result['js'] = $from_tree['js'];

        return $result;
    }

    /**
     * @param array<int, array<string, mixed>> $children
     * @return array{css: array<int, string>, js: array<int, string>}
     */
    private function extract_assets_from_page_children(array $children): array
    {
        $result = ['css' => [], 'js' => []];

        foreach ($children as $child) {
            if (!is_array($child)) {
                continue;
            }
            $type = strtolower((string) ($child['type'] ?? ''));
            if ($type !== 'assets') {
                continue;
            }

            $asset_nodes = is_array($child['children'] ?? null) ? $child['children'] : [];
            foreach ($asset_nodes as $asset_node) {
                if (!is_array($asset_node)) {
                    continue;
                }
                $asset_type = strtolower((string) ($asset_node['type'] ?? ''));
                $nested = is_array($asset_node['children'] ?? null) ? $asset_node['children'] : [];

                if ($asset_type === 'head') {
                    $collected = $this->extract_template_assets_from_nodes($nested, 'css');
                    $result['css'] = array_merge($result['css'], $collected);
                    continue;
                }

                if ($asset_type === 'foot') {
                    $collected = $this->extract_template_assets_from_nodes($nested, 'js');
                    $result['js'] = array_merge($result['js'], $collected);
                    continue;
                }

                // Allow direct template nodes under Assets as a shorthand.
                $direct = $this->extract_template_assets_from_nodes([$asset_node], '');
                $result['css'] = array_merge($result['css'], $direct['css']);
                $result['js'] = array_merge($result['js'], $direct['js']);
            }
        }

        $result['css'] = array_values(array_unique($result['css']));
        $result['js'] = array_values(array_unique($result['js']));
        return $result;
    }

    /**
     * @param array<int, array<string, mixed>> $nodes
     * @param string $sectionHint css|js|''
     * @return array<int, string>|array{css: array<int, string>, js: array<int, string>}
     */
    private function extract_template_assets_from_nodes(array $nodes, string $sectionHint): array
    {
        $as_lists = $sectionHint === '';
        $css = [];
        $js = [];

        foreach ($nodes as $node) {
            if (!is_array($node)) {
                continue;
            }
            $type = strtolower((string) ($node['type'] ?? ''));
            $props = is_array($node['props'] ?? null) ? $node['props'] : [];
            $name = isset($props['name']) ? (string) $props['name'] : '';
            $normalized = $this->normalize_asset_list($name);
            if ($normalized === []) {
                continue;
            }

            if ($type === 'csstemplate' || ($sectionHint === 'css' && $type === 'template')) {
                $css = array_merge($css, $normalized);
            }
            if ($type === 'jstemplate' || ($sectionHint === 'js' && $type === 'template')) {
                $js = array_merge($js, $normalized);
            }
        }

        $css = array_values(array_unique($css));
        $js = array_values(array_unique($js));

        if ($as_lists) {
            return ['css' => $css, 'js' => $js];
        }

        return $sectionHint === 'css' ? $css : $js;
    }

    /**
     * @return array<int, string>
     */
    private function normalize_asset_list(mixed $value): array
    {
        $items = [];
        if (is_array($value)) {
            $items = $value;
        } elseif (is_string($value) || is_numeric($value)) {
            $items = [$value];
        }

        $out = [];
        foreach ($items as $item) {
            $key = strtolower(trim((string) $item));
            $key = preg_replace('/[^a-z0-9_-]/', '', $key) ?: '';
            if ($key === '' || !isset(self::ASSET_REGISTRY[$key])) {
                continue;
            }
            $out[] = $key;
        }

        return array_values(array_unique($out));
    }

    private function enqueue_registered_asset(string $key, string $type): void
    {
        if (!isset(self::ASSET_REGISTRY[$key])) {
            return;
        }

        $asset = self::ASSET_REGISTRY[$key];
        $deps = is_array($asset['deps'] ?? null) ? $asset['deps'] : [];

        if ($type === 'css' && isset($asset['css'])) {
            wp_enqueue_style('sml-ext-' . $key . '-css', $asset['css'], $deps, null);
            return;
        }

        if ($type === 'js' && isset($asset['js'])) {
            wp_enqueue_script('sml-ext-' . $key . '-js', $asset['js'], $deps, null, true);
        }
    }

    private static function sanitize_template_filename(string $name): string
    {
        $name = strtolower(trim(str_replace('\\', '/', $name)));
        $name = basename($name);
        $name = preg_replace('/[^a-z0-9._-]/', '', $name) ?: '';
        if ($name === '' || !str_ends_with($name, '.twig')) {
            return '';
        }

        return $name;
    }

    private static function sanitize_markdown_filename(string $name): string
    {
        $name = strtolower(trim(str_replace('\\', '/', $name)));
        $name = basename($name);
        $name = preg_replace('/[^a-z0-9._-]/', '', $name) ?: '';
        if ($name === '' || !str_ends_with($name, '.md')) {
            return '';
        }

        return $name;
    }

}

$GLOBALS['sml_pages_plugin'] = new SML_Pages_Plugin();

function sml_pages_plugin_activate(): void
{
    $plugin = new SML_Pages_Plugin();
    $plugin->register_post_type();
    flush_rewrite_rules();
}
register_activation_hook(__FILE__, 'sml_pages_plugin_activate');

function sml_pages_plugin_deactivate(): void
{
    flush_rewrite_rules();
}
register_deactivation_hook(__FILE__, 'sml_pages_plugin_deactivate');

function sml_pages_get_markdown_part_content(string $part): ?string
{
    return SML_Pages_Plugin::get_markdown_part_content($part);
}

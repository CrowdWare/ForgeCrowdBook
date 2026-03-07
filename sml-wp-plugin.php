<?php
/**
 * Plugin Name: SML Pages MVP
 * Description: Monaco-ready SML page pipeline for WordPress (Page, Row, Column, Card, Link, Markdown, Image, Spacer).
 * Version: 0.1.0
 * Author: Art
 */

if (!defined('ABSPATH')) {
    exit;
}

require_once __DIR__ . '/includes/class-sml-parser.php';
require_once __DIR__ . '/includes/class-sml-renderer.php';

class SML_Pages_Plugin
{
    public const META_SOURCE = '_sml_source';
    public const META_RENDERED = '_sml_rendered_html';
    public const META_TEMPLATE_MODE = '_sml_template_mode';

    public function __construct()
    {
        add_action('init', [$this, 'register_post_type']);
        add_action('add_meta_boxes', [$this, 'add_meta_boxes']);
        add_action('save_post_sml_page', [$this, 'save_sml_page'], 10, 2);
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

        echo '<p>Use your SML DSL here. Supported: Page, Row, Column, Card, Link, Markdown, Image, Spacer.</p>';
        echo '<p><label for="sml_template_mode"><strong>Template Mode:</strong></label> ';
        echo '<select id="sml_template_mode" name="sml_template_mode">';
        echo '<option value="canvas"' . selected($template_mode, 'canvas', false) . '>Canvas (full viewport)</option>';
        echo '<option value="theme"' . selected($template_mode, 'theme', false) . '>Theme (with header/footer)</option>';
        echo '</select></p>';
        echo '<div id="sml_monaco_editor" aria-label="SML Monaco Editor"></div>';
        echo '<textarea id="sml_source" name="sml_source" style="width:100%;min-height:380px;font-family:monospace;">' . esc_textarea($source) . '</textarea>';
        echo '<p><small>Markdown supports <code>text</code> or <code>part</code>. Parts are loaded from <code>wp-content/uploads/sml-parts/</code>.</small></p>';
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
    }

    private function compile_source(string $source): string
    {
        try {
            $parser = new SML_Parser();
            $nodes = $parser->parse($source);

            $renderer = new SML_Renderer();
            return $renderer->render($nodes);
        } catch (Throwable $e) {
            return '<pre class="sml-error">Compile error: ' . esc_html($e->getMessage()) . '</pre>';
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
        if (!$screen || $screen->post_type !== 'sml_page') {
            return;
        }

        wp_enqueue_style('sml-admin', plugins_url('assets/sml-admin.css', __FILE__), [], '0.1.0');
        wp_enqueue_script('sml-monaco-loader', 'https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.52.2/min/vs/loader.min.js', [], null, true);
        wp_enqueue_script('sml-admin', plugins_url('assets/sml-admin.js', __FILE__), ['sml-monaco-loader'], '0.1.0', true);

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

        wp_enqueue_style('sml-pico', 'https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css', [], null);
        wp_enqueue_style('sml-frontend', plugins_url('assets/sml-frontend.css', __FILE__), ['sml-pico'], '0.1.0');
    }

    public static function get_rendered_for_post(int $post_id): string
    {
        return (string) get_post_meta($post_id, self::META_RENDERED, true);
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

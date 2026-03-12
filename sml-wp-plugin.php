<?php
/**
 * Plugin Name: Forge WP SML Compiler
 * Plugin URI: https://codeberg.org/CrowdWare/ForgeCrowdBook
 * Description: SML Compiler for WordPress: build pages with SML, Twig and Markdown, then ship super fast static HTML.
 * Version: 0.1.64
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
$crowdbook_boot_file = __DIR__ . '/crowdbook/crowdbook.php';
if (is_readable($crowdbook_boot_file)) {
    require_once $crowdbook_boot_file;
}

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
    public const OPTION_LANDING_PAGE_ID = 'sml_landing_page_id';
    private bool $is_syncing_front_page = false;

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
        add_action('admin_init', [$this, 'register_landing_page_setting']);
        add_action('add_meta_boxes', [$this, 'add_meta_boxes']);
        add_action('save_post_sml_page', [$this, 'save_sml_page'], 10, 2);
        add_action('save_post_sml_template', [$this, 'save_sml_template'], 10, 2);
        add_action('save_post_sml_markdown_part', [$this, 'save_sml_markdown_part'], 10, 2);
        add_action('template_redirect', [$this, 'maybe_render_landing_page'], 0);
        add_action('template_redirect', [$this, 'maybe_redirect_landing_permalink_to_home'], 1);
        add_action('template_redirect', [$this, 'maybe_redirect_legacy_sml_base_url'], 2);
        add_action('template_redirect', [$this, 'maybe_render_sml_page_on_404'], 3);
        add_action('parse_request', [$this, 'resolve_root_sml_page_request'], 20);
        add_action('admin_menu', [$this, 'register_help_submenu']);
        add_filter('template_include', [$this, 'template_include']);
        add_filter('wp_dropdown_pages', [$this, 'inject_sml_pages_into_frontpage_dropdown'], 10, 2);
        add_filter('post_type_link', [$this, 'filter_sml_page_permalink'], 10, 3);
        add_shortcode('sml_page', [$this, 'shortcode_sml_page']);
        add_action('update_option_' . self::OPTION_LANDING_PAGE_ID, [$this, 'sync_wp_front_page_from_sml_option'], 10, 2);
        add_action('update_option_page_on_front', [$this, 'sync_sml_option_from_wp_front_page'], 10, 2);

        add_action('admin_enqueue_scripts', [$this, 'admin_assets']);
        add_action('wp_enqueue_scripts', [$this, 'frontend_assets']);
    }

    public function register_help_submenu(): void
    {
        add_submenu_page(
            'edit.php?post_type=sml_page',
            'SML Hilfe',
            'SML Hilfe',
            'edit_posts',
            'sml-help',
            [$this, 'render_help_page']
        );
    }

    public function render_help_page(): void
    {
        if (!current_user_can('edit_posts')) {
            wp_die('Insufficient permissions.');
        }

        echo '<div class="wrap">';
        echo '<h1>SML Hilfe</h1>';
        echo '<p>Referenz fuer SML-Elemente, Properties und Beispiele.</p>';

        echo '<h2>Schnellstart</h2>';
        echo '<pre style="background:#1e1e1e;color:#f6f8fa;padding:12px;overflow:auto;">';
        echo esc_html("Page {\n  padding: 32\n  bgColor: \"#2D2D2D\"\n  color: \"#ffffff\"\n  Column {\n    gap: 16\n    Markdown { text: \"# Hallo SML\" }\n    Link { href: \"/test\" text: \"Mehr lesen\" }\n  }\n}");
        echo '</pre>';
        echo '<p><strong>Wichtig:</strong> Bei Farben immer gueltige CSS-Farben nutzen, z. B. <code>#2D2D2D</code>, <code>rgb(45,45,45)</code>, <code>white</code>.</p>';

        echo '<h2>Gemeinsame Style-Properties</h2>';
        echo '<table class="widefat striped" style="max-width:980px">';
        echo '<thead><tr><th>Property</th><th>Typ</th><th>Beispiel</th><th>Hinweis</th></tr></thead><tbody>';
        echo '<tr><td><code>padding</code></td><td>Zahl / String / Liste</td><td><code>padding: 32</code>, <code>padding: \"2rem\"</code>, <code>padding: 12, 24</code></td><td>Zahlen werden als <code>px</code> interpretiert.</td></tr>';
        echo '<tr><td><code>bgColor</code></td><td>String</td><td><code>bgColor: \"#2D2D2D\"</code></td><td>Hex, rgb/rgba, hsl/hsla oder Farbname.</td></tr>';
        echo '<tr><td><code>color</code></td><td>String</td><td><code>color: \"#fff\"</code></td><td>Textfarbe.</td></tr>';
        echo '<tr><td><code>gap</code></td><td>Zahl / String</td><td><code>gap: 16</code></td><td>Setzt Abstand zwischen Kindern.</td></tr>';
        echo '<tr><td><code>scrollable</code></td><td>Boolean</td><td><code>scrollable: true</code></td><td>Setzt <code>overflow:auto</code>.</td></tr>';
        echo '<tr><td><code>class</code> / <code>classes</code></td><td>String / Liste</td><td><code>class: \"my-block\"</code></td><td>Zusaetzliche CSS-Klassen.</td></tr>';
        echo '</tbody></table>';

        echo '<h2>Elemente und Properties</h2>';
        echo '<table class="widefat striped" style="max-width:980px">';
        echo '<thead><tr><th>Element</th><th>Properties</th><th>Beispiel</th></tr></thead><tbody>';
        echo '<tr><td><code>Page</code></td><td>Gemeinsame Style-Properties, Kinder</td><td><code>Page { padding: 24 Column { ... } }</code></td></tr>';
        echo '<tr><td><code>Row</code></td><td>Gemeinsame Style-Properties, Kinder</td><td><code>Row { gap: 12 Card { ... } Card { ... } }</code></td></tr>';
        echo '<tr><td><code>Column</code></td><td>Gemeinsame Style-Properties, Kinder</td><td><code>Column { gap: 16 Markdown { text: \"...\" } }</code></td></tr>';
        echo '<tr><td><code>Hero</code></td><td>Wie Container, Kinder</td><td><code>Hero { padding: 32 Markdown { text: \"# Titel\" } }</code></td></tr>';
        echo '<tr><td><code>Card</code></td><td><code>title</code>, <code>subtitle</code>, gemeinsame Styles, Kinder</td><td><code>Card { title: \"Titel\" subtitle: \"Sub\" Markdown { text: \"Inhalt\" } }</code></td></tr>';
        echo '<tr><td><code>Link</code></td><td><code>href</code>, <code>text</code>, <code>target</code>, gemeinsame Styles, Kinder</td><td><code>Link { href: \"/books\" text: \"Zu den Buechern\" target: \"_blank\" }</code></td></tr>';
        echo '<tr><td><code>Markdown</code></td><td><code>text</code> oder <code>part</code>, gemeinsame Styles</td><td><code>Markdown { text: \"# Ueberschrift\" }</code> / <code>Markdown { part: \"home.md\" }</code></td></tr>';
        echo '<tr><td><code>Image</code></td><td><code>src</code>, <code>alt</code>, <code>width</code>, <code>height</code>, gemeinsame Styles</td><td><code>Image { src: \"https://.../bild.jpg\" alt: \"Beschreibung\" width: 640 }</code></td></tr>';
        echo '<tr><td><code>Spacer</code></td><td><code>amount</code></td><td><code>Spacer { amount: 24 }</code></td></tr>';
        echo '<tr><td><code>Assets</code></td><td>Kinder: <code>Head</code>, <code>Foot</code>, <code>CssTemplate</code>, <code>JsTemplate</code></td><td><code>Assets { Head { CssTemplate { name: \"bootstrap\" } } Foot { JsTemplate { name: \"bootstrap\" } } }</code></td></tr>';
        echo '<tr><td><code>When</code></td><td>Switch fuer Responsive-Cases</td><td><code>When { Desktop { ... } MobilePortrait { ... } MobileLandscape { ... } Default { ... } }</code></td></tr>';
        echo '<tr><td><code>IncludeSml</code></td><td><code>part</code> oder <code>src</code></td><td><code>IncludeSml { part: \"content-mobile.sml\" }</code></td></tr>';
        echo '</tbody></table>';

        echo '<h2>Responsive-Pattern (empfohlen)</h2>';
        echo '<pre style="background:#1e1e1e;color:#f6f8fa;padding:12px;overflow:auto;">';
        echo esc_html("Page {\n  When {\n    Desktop {\n      IncludeSml { part: \"content-desktop.sml\" }\n    }\n    MobilePortrait {\n      IncludeSml { part: \"content-mobile-portrait.sml\" }\n    }\n    MobileLandscape {\n      IncludeSml { part: \"content-mobile-landscape.sml\" }\n    }\n    Default {\n      Markdown { text: \"Fallback\" }\n    }\n  }\n}");
        echo '</pre>';
        echo '<p><small>SML-Part-Dateien werden in <code>wp-content/uploads/sml-parts/</code> gesucht (oder aus den Markdown-Files, wenn dort der Name existiert).</small></p>';

        echo '<h2>Unterstuetzte Asset-Namen</h2>';
        echo '<p><code>bootstrap</code>, <code>tailwind</code>, <code>pico</code></p>';

        echo '<h2>Hinweise</h2>';
        echo '<ul>';
        echo '<li>Strings immer in Anfuehrungszeichen schreiben: <code>\"...\"</code>.</li>';
        echo '<li>Zwischen Properties sind keine Kommas noetig; Zeilenumbrueche reichen.</li>';
        echo '<li>Der Root-Knoten sollte <code>Page { ... }</code> sein.</li>';
        echo '</ul>';

        echo '</div>';
    }

    public function register_landing_page_setting(): void
    {
        register_setting('reading', self::OPTION_LANDING_PAGE_ID, [
            'type' => 'integer',
            'sanitize_callback' => [$this, 'sanitize_landing_page_setting'],
            'default' => 0,
        ]);

        add_settings_field(
            self::OPTION_LANDING_PAGE_ID,
            'SML Landing Page',
            [$this, 'render_landing_page_setting'],
            'reading'
        );
    }

    public function sanitize_landing_page_setting($value): int
    {
        $id = (int) $value;
        if ($id <= 0) {
            return 0;
        }

        $post = get_post($id);
        if (!($post instanceof WP_Post)) {
            return 0;
        }

        if ($post->post_type !== 'sml_page') {
            return 0;
        }

        if ($post->post_status !== 'publish') {
            return 0;
        }

        return $id;
    }

    public function render_landing_page_setting(): void
    {
        $selected = (int) get_option(self::OPTION_LANDING_PAGE_ID, 0);
        if ($selected <= 0) {
            $front_id = (int) get_option('page_on_front', 0);
            if ($front_id > 0 && get_post_type($front_id) === 'sml_page') {
                $selected = $front_id;
            }
        }
        $pages = get_posts([
            'post_type' => 'sml_page',
            'post_status' => ['publish'],
            'numberposts' => -1,
            'orderby' => 'title',
            'order' => 'ASC',
        ]);

        echo '<select id="' . esc_attr(self::OPTION_LANDING_PAGE_ID) . '" name="' . esc_attr(self::OPTION_LANDING_PAGE_ID) . '">';
        echo '<option value="0">' . esc_html__('— Keine —', 'default') . '</option>';
        foreach ($pages as $page) {
            echo '<option value="' . (int) $page->ID . '"' . selected($selected, (int) $page->ID, false) . '>' . esc_html($page->post_title ?: ('#' . (int) $page->ID)) . '</option>';
        }
        echo '</select>';
        echo '<p class="description">Wenn gesetzt, wird diese SML-Seite auf der Start-URL (<code>/</code>) als Landing-Page gerendert.</p>';
    }

    public function inject_sml_pages_into_frontpage_dropdown(string $output, array $args): string
    {
        if (!$this->is_front_page_dropdown_args($args)) {
            return $output;
        }

        if (!str_contains($output, '</select>')) {
            return $output;
        }

        $selected = isset($args['selected']) ? (int) $args['selected'] : 0;
        $pages = get_posts([
            'post_type' => 'sml_page',
            'post_status' => ['publish'],
            'numberposts' => -1,
            'orderby' => 'title',
            'order' => 'ASC',
        ]);

        if (!is_array($pages) || $pages === []) {
            return $output;
        }

        $extra = '<optgroup label="' . esc_attr__('SML Pages', 'default') . '">';
        foreach ($pages as $page) {
            $title = (string) $page->post_title;
            if ($title === '') {
                $title = '#' . (int) $page->ID;
            }
            $extra .= '<option class="level-0" value="' . (int) $page->ID . '"' . selected($selected, (int) $page->ID, false) . '>' . esc_html($title) . '</option>';
        }
        $extra .= '</optgroup>';

        return str_replace('</select>', $extra . '</select>', $output);
    }

    public function sync_wp_front_page_from_sml_option($old_value, $new_value): void
    {
        if ($this->is_syncing_front_page) {
            return;
        }

        $new_id = (int) $new_value;
        if ($new_id <= 0) {
            return;
        }

        $post = get_post($new_id);
        if (!($post instanceof WP_Post) || $post->post_type !== 'sml_page' || $post->post_status !== 'publish') {
            return;
        }

        $this->is_syncing_front_page = true;
        update_option('show_on_front', 'page');
        update_option('page_on_front', $new_id);
        $this->is_syncing_front_page = false;
    }

    public function sync_sml_option_from_wp_front_page($old_value, $new_value): void
    {
        if ($this->is_syncing_front_page) {
            return;
        }

        $new_id = (int) $new_value;
        $this->is_syncing_front_page = true;
        if ($new_id > 0 && get_post_type($new_id) === 'sml_page') {
            update_option(self::OPTION_LANDING_PAGE_ID, $new_id);
        } elseif ((int) get_option(self::OPTION_LANDING_PAGE_ID, 0) === (int) $old_value) {
            update_option(self::OPTION_LANDING_PAGE_ID, 0);
        }
        $this->is_syncing_front_page = false;
    }

    private function is_front_page_dropdown_args(array $args): bool
    {
        $name = isset($args['name']) ? (string) $args['name'] : '';
        $id = isset($args['id']) ? (string) $args['id'] : '';

        if ($name === 'page_on_front' || $id === 'page_on_front') {
            return true;
        }

        return str_contains($name, 'page_on_front');
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
            'has_archive' => false,
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

        $this->register_root_rewrites_for_sml_pages();
    }

    private function register_root_rewrites_for_sml_pages(): void
    {
        $items = get_posts([
            'post_type' => 'sml_page',
            'post_status' => ['publish'],
            'numberposts' => -1,
            'fields' => 'ids',
        ]);
        if (!is_array($items) || $items === []) {
            return;
        }

        foreach ($items as $id) {
            $id = (int) $id;
            if ($id <= 0) {
                continue;
            }

            $slug = (string) get_post_field('post_name', $id);
            $slug = sanitize_title($slug);
            if ($slug === '') {
                continue;
            }

            add_rewrite_rule('^' . preg_quote($slug, '#') . '/?$', 'index.php?post_type=sml_page&p=' . $id, 'top');
        }
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

        echo '<p><label for="sml_markdown_part_name"><strong>Part File Name</strong> (e.g. <code>content-mobile.sml</code> or <code>home.md</code>)</label></p>';
        echo '<input type="text" id="sml_markdown_part_name" name="sml_markdown_part_name" value="' . esc_attr($name) . '" style="width:100%;max-width:420px;" />';
        echo '<p><small>Use this with <code>Markdown { part: "' . esc_html($name) . '" }</code> or <code>IncludeSml { part: "' . esc_html($name) . '" }</code>.</small></p>';

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

        // If a source page changes, all pages with IncludeSml should refresh automatically.
        $this->recompile_pages_with_include_sml();
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
            $name = 'part-' . $post_id . '.sml';
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

    private function recompile_pages_with_include_sml(): void
    {
        $this->recompile_matching_pages(static function (string $source): bool {
            return stripos($source, 'IncludeSml') !== false;
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
            $custom = $this->resolve_sml_template_file($post_id);
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

        wp_enqueue_style('sml-admin', plugins_url('assets/sml-admin.css', __FILE__), [], '0.1.64');
        wp_enqueue_script('sml-monaco-loader', 'https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.52.2/min/vs/loader.min.js', [], null, true);
        wp_enqueue_script('sml-admin', plugins_url('assets/sml-admin.js', __FILE__), ['sml-monaco-loader'], '0.1.64', true);

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
        $post_id = 0;
        if (is_singular('sml_page')) {
            $post_id = (int) get_queried_object_id();
        } elseif ($this->is_front_root_request()) {
            $post_id = (int) get_option(self::OPTION_LANDING_PAGE_ID, 0);
        }

        if ($post_id <= 0 || get_post_type($post_id) !== 'sml_page') {
            return;
        }

        wp_enqueue_style('sml-frontend', plugins_url('assets/sml-frontend.css', __FILE__), [], '0.1.50');
        $this->enqueue_page_assets_for_post($post_id);
    }

    private function enqueue_page_assets_for_post(int $post_id): void
    {
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

    private function force_non_404_query_state(WP_Post $post): void
    {
        global $wp_query;

        if ($wp_query instanceof WP_Query) {
            $wp_query->is_404 = false;
            $wp_query->is_singular = true;
            $wp_query->is_single = false;
            $wp_query->is_page = false;
            $wp_query->is_home = false;
            $wp_query->is_archive = false;
            $wp_query->is_posts_page = false;
            $wp_query->queried_object = $post;
            $wp_query->queried_object_id = (int) $post->ID;
        }
    }

    public function maybe_render_landing_page(): void
    {
        if (!$this->is_front_root_request()) {
            return;
        }

        if (is_singular('sml_page')) {
            return;
        }

        $post_id = (int) get_option(self::OPTION_LANDING_PAGE_ID, 0);
        if ($post_id <= 0) {
            return;
        }

        $landing_post = get_post($post_id);
        if (!($landing_post instanceof WP_Post) || $landing_post->post_type !== 'sml_page' || $landing_post->post_status !== 'publish') {
            return;
        }

        $template = $this->resolve_sml_template_file($post_id);
        if (!is_file($template)) {
            return;
        }

        status_header(200);
        nocache_headers();
        $this->force_non_404_query_state($landing_post);
        wp_enqueue_style('sml-frontend', plugins_url('assets/sml-frontend.css', __FILE__), [], '0.1.50');

        global $post;
        $previous_post = $post;
        $post = $landing_post;
        setup_postdata($landing_post);
        include $template;
        wp_reset_postdata();
        $post = $previous_post;
        exit;
    }

    public function maybe_redirect_landing_permalink_to_home(): void
    {
        if (!is_singular('sml_page')) {
            return;
        }

        if (is_preview()) {
            return;
        }

        $landing_id = (int) get_option(self::OPTION_LANDING_PAGE_ID, 0);
        if ($landing_id <= 0) {
            return;
        }

        $current_id = (int) get_queried_object_id();
        if ($current_id !== $landing_id) {
            return;
        }

        $home = home_url('/');
        $current = (string) get_permalink($current_id);
        if ($current === '' || untrailingslashit($current) === untrailingslashit($home)) {
            return;
        }

        wp_safe_redirect($home, 301);
        exit;
    }

    public function maybe_redirect_legacy_sml_base_url(): void
    {
        if (is_admin() || wp_doing_ajax()) {
            return;
        }

        $path = isset($_SERVER['REQUEST_URI']) ? (string) parse_url((string) $_SERVER['REQUEST_URI'], PHP_URL_PATH) : '';
        $path = trim($path, '/');
        if ($path === '' || !str_starts_with($path, 'sml/')) {
            return;
        }

        if (!preg_match('#^sml/([^/]+)/?$#', $path, $m)) {
            return;
        }

        $slug = sanitize_title((string) $m[1]);
        if ($slug === '') {
            return;
        }

        $post = get_page_by_path($slug, OBJECT, 'sml_page');
        if (!($post instanceof WP_Post)) {
            return;
        }

        $target = get_permalink((int) $post->ID);
        if (!is_string($target) || $target === '') {
            return;
        }

        $current_path = isset($_SERVER['REQUEST_URI']) ? (string) parse_url((string) $_SERVER['REQUEST_URI'], PHP_URL_PATH) : '';
        $current_url = home_url($current_path);
        if (untrailingslashit($target) === untrailingslashit($current_url)) {
            return;
        }

        wp_safe_redirect($target, 301);
        exit;
    }

    public function maybe_render_sml_page_on_404(): void
    {
        if (is_admin() || wp_doing_ajax()) {
            return;
        }

        if (defined('REST_REQUEST') && REST_REQUEST) {
            return;
        }

        if (!is_404()) {
            return;
        }

        $path = isset($_SERVER['REQUEST_URI']) ? (string) parse_url((string) $_SERVER['REQUEST_URI'], PHP_URL_PATH) : '';
        $home_path = (string) parse_url(home_url('/'), PHP_URL_PATH);
        $path = trim((string) preg_replace('#^' . preg_quote((string) $home_path, '#') . '#', '', $path), '/');
        if ($path === '' || str_contains($path, '/')) {
            return;
        }

        $slug = sanitize_title($path);
        if ($slug === '') {
            return;
        }

        $landing_id = (int) get_option(self::OPTION_LANDING_PAGE_ID, 0);
        $target_post = get_page_by_path($slug, OBJECT, 'sml_page');
        if (!($target_post instanceof WP_Post) || $target_post->post_status !== 'publish' || (int) $target_post->ID === $landing_id) {
            return;
        }

        $template = $this->resolve_sml_template_file((int) $target_post->ID);
        if (!is_file($template)) {
            return;
        }

        status_header(200);
        nocache_headers();
        $this->force_non_404_query_state($target_post);
        wp_enqueue_style('sml-frontend', plugins_url('assets/sml-frontend.css', __FILE__), [], '0.1.50');
        $this->enqueue_page_assets_for_post((int) $target_post->ID);

        global $post;
        $previous_post = $post;
        $post = $target_post;
        setup_postdata($target_post);
        include $template;
        wp_reset_postdata();
        $post = $previous_post;
        exit;
    }

    public function filter_sml_page_permalink(string $permalink, WP_Post $post, bool $leavename): string
    {
        if ($post->post_type !== 'sml_page') {
            return $permalink;
        }

        $slug = sanitize_title((string) $post->post_name);
        if ($slug === '') {
            return $permalink;
        }

        $landing_id = (int) get_option(self::OPTION_LANDING_PAGE_ID, 0);
        if ($landing_id > 0 && (int) $post->ID === $landing_id) {
            return home_url('/');
        }

        return home_url('/' . $slug . '/');
    }

    public function resolve_root_sml_page_request(WP $wp): void
    {
        if (is_admin() || wp_doing_ajax()) {
            return;
        }

        if (defined('REST_REQUEST') && REST_REQUEST) {
            return;
        }

        if (!empty($wp->query_vars)) {
            return;
        }

        $path = isset($_SERVER['REQUEST_URI']) ? (string) parse_url((string) $_SERVER['REQUEST_URI'], PHP_URL_PATH) : '';
        $home_path = (string) parse_url(home_url('/'), PHP_URL_PATH);
        $path = trim((string) preg_replace('#^' . preg_quote((string) $home_path, '#') . '#', '', $path), '/');
        if ($path === '' || str_contains($path, '/')) {
            return;
        }

        // Skip common WP/system namespaces.
        if (in_array($path, ['wp-admin', 'wp-login.php', 'wp-json', 'feed', 'comments'], true)) {
            return;
        }

        $slug = sanitize_title($path);
        if ($slug === '') {
            return;
        }

        $post = get_page_by_path($slug, OBJECT, 'sml_page');
        if (!($post instanceof WP_Post) || $post->post_status !== 'publish') {
            return;
        }

        $wp->query_vars = [
            'post_type' => 'sml_page',
            'name' => $slug,
        ];
    }

    private function resolve_sml_template_file(int $post_id): string
    {
        $mode = (string) get_post_meta($post_id, self::META_TEMPLATE_MODE, true);
        if (!in_array($mode, ['theme', 'canvas'], true)) {
            $mode = 'canvas';
        }

        return ($mode === 'theme')
            ? __DIR__ . '/templates/single-sml_page.php'
            : __DIR__ . '/templates/single-sml_page-canvas.php';
    }

    private function is_front_root_request(): bool
    {
        if (is_admin() || wp_doing_ajax()) {
            return false;
        }

        if (defined('REST_REQUEST') && REST_REQUEST) {
            return false;
        }

        $uri_path = isset($_SERVER['REQUEST_URI']) ? (string) parse_url((string) $_SERVER['REQUEST_URI'], PHP_URL_PATH) : '';
        $home_path = (string) parse_url(home_url('/'), PHP_URL_PATH);
        $uri_path = trim($uri_path, '/');
        $home_path = trim($home_path, '/');

        if ($uri_path !== $home_path) {
            return false;
        }

        return true;
    }

    public static function get_rendered_for_post(int $post_id): string
    {
        if ($post_id <= 0) {
            return '';
        }

        $html = (string) get_post_meta($post_id, self::META_RENDERED, true);
        if (trim($html) !== '') {
            return $html;
        }

        $source = (string) get_post_meta($post_id, self::META_SOURCE, true);
        if (trim($source) === '') {
            return '';
        }

        try {
            $parser = new SML_Parser();
            $nodes = $parser->parse($source);
            $global_templates = self::get_global_template_overrides();
            $renderer = new SML_Renderer(__DIR__ . '/templates', null, $global_templates);
            $html = $renderer->render($nodes);
            update_post_meta($post_id, self::META_RENDERED, $html);
            return $html;
        } catch (Throwable $e) {
            return '<pre class="sml-error">Compile error: ' . esc_html($e->getMessage()) . '</pre>';
        }
    }

    public static function get_theme_wrapper_style_for_post(int $post_id): string
    {
        if ($post_id <= 0) {
            return '';
        }

        $source = (string) get_post_meta($post_id, self::META_SOURCE, true);
        if (trim($source) === '') {
            return '';
        }

        try {
            $parser = new SML_Parser();
            $nodes = $parser->parse($source);
        } catch (Throwable) {
            return '';
        }

        $first = $nodes[0] ?? null;
        if (!is_array($first) || strtolower((string) ($first['type'] ?? '')) !== 'page') {
            return '';
        }

        $props = is_array($first['props'] ?? null) ? $first['props'] : [];
        $styles = [];

        $bg = self::sanitize_help_css_color((string) ($props['wrapperBgColor'] ?? $props['bgColor'] ?? ''));
        if ($bg !== '') {
            $styles[] = 'background-color:' . $bg;
            $styles[] = '--sml-wrapper-bg:' . $bg;
        }

        $color = self::sanitize_help_css_color((string) ($props['wrapperColor'] ?? $props['color'] ?? ''));
        if ($color !== '') {
            $styles[] = 'color:' . $color;
            $styles[] = '--sml-wrapper-color:' . $color;
        }

        if (array_key_exists('wrapperPadding', $props)) {
            $styles[] = 'padding:' . self::help_spacing_value($props['wrapperPadding']);
        } elseif (array_key_exists('padding', $props)) {
            $styles[] = 'padding:' . self::help_spacing_value($props['padding']);
        }

        return implode(';', array_filter($styles, static fn($v) => is_string($v) && $v !== ''));
    }

    private static function help_spacing_value(mixed $value): string
    {
        if (is_array($value)) {
            $parts = array_map(static fn($v) => self::help_numeric_unit($v), $value);
            return implode(' ', array_slice($parts, 0, 4));
        }

        return self::help_numeric_unit($value);
    }

    private static function help_numeric_unit(mixed $value): string
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

    private static function sanitize_help_css_color(string $value): string
    {
        $value = trim($value);
        if ($value === '') {
            return '';
        }

        if (preg_match('/^#(?:[A-Fa-f0-9]{3}|[A-Fa-f0-9]{4}|[A-Fa-f0-9]{6}|[A-Fa-f0-9]{8})$/', $value)) {
            return $value;
        }

        if (preg_match('/^(?:rgb|rgba|hsl|hsla)\\([^\\)]+\\)$/i', $value)) {
            return $value;
        }

        if (preg_match('/^[a-zA-Z]+$/', $value)) {
            return strtolower($value);
        }

        return '';
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
        if ($name === '') {
            return '';
        }

        if (!str_ends_with($name, '.md') && !str_ends_with($name, '.sml')) {
            return '';
        }

        return $name;
    }

}

$GLOBALS['sml_pages_plugin'] = new SML_Pages_Plugin();
$GLOBALS['crowdbook_plugin'] = null;

if (class_exists('CrowdBook_Plugin')) {
    try {
        $GLOBALS['crowdbook_plugin'] = new CrowdBook_Plugin();
    } catch (Throwable $e) {
        error_log('CrowdBook bootstrap failed: ' . $e->getMessage());
    }
}

function sml_pages_plugin_activate(): void
{
    $plugin = new SML_Pages_Plugin();
    $plugin->register_post_type();

    if (class_exists('CrowdBook_Plugin')) {
        try {
            CrowdBook_Plugin::activate();
        } catch (Throwable $e) {
            error_log('CrowdBook activation failed: ' . $e->getMessage());
        }
    }

    flush_rewrite_rules();
}
register_activation_hook(__FILE__, 'sml_pages_plugin_activate');

function sml_pages_plugin_deactivate(): void
{
    if (class_exists('CrowdBook_Plugin')) {
        CrowdBook_Plugin::deactivate();
    }

    flush_rewrite_rules();
}
register_deactivation_hook(__FILE__, 'sml_pages_plugin_deactivate');

function sml_pages_get_markdown_part_content(string $part): ?string
{
    return SML_Pages_Plugin::get_markdown_part_content($part);
}

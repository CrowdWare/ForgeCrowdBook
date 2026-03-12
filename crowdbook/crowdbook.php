<?php

if (!defined('ABSPATH')) {
    exit;
}

require_once __DIR__ . '/includes/class-mailer.php';
require_once __DIR__ . '/includes/class-books.php';
require_once __DIR__ . '/includes/class-users.php';
require_once __DIR__ . '/includes/class-auth.php';
require_once __DIR__ . '/includes/class-spam-filter.php';
require_once __DIR__ . '/includes/class-sml-renderer.php';
require_once __DIR__ . '/includes/class-chapters.php';
require_once __DIR__ . '/includes/class-likes.php';
require_once __DIR__ . '/includes/class-social.php';

require_once __DIR__ . '/frontend/auth.php';
require_once __DIR__ . '/frontend/login.php';
require_once __DIR__ . '/frontend/register.php';
require_once __DIR__ . '/frontend/dashboard.php';
require_once __DIR__ . '/frontend/editor.php';
require_once __DIR__ . '/frontend/book-index.php';
require_once __DIR__ . '/frontend/books.php';

require_once __DIR__ . '/admin/chapters-page.php';
require_once __DIR__ . '/admin/users-page.php';
require_once __DIR__ . '/admin/dashboard-page.php';
require_once __DIR__ . '/admin/books-page.php';

class CrowdBook_Plugin
{
    private const DB_VERSION = '3.4.1';
    private const CAP_MODERATE = 'moderate_crowdbook';

    private CrowdBook_Mailer $mailer;
    private CrowdBook_Books $books;
    private CrowdBook_Users $users;
    private CrowdBook_Auth $auth;
    private CrowdBook_Spam_Filter $spam_filter;
    private CrowdBook_SML_Renderer $sml_renderer;
    private CrowdBook_Chapters $chapters;
    private CrowdBook_Likes $likes;
    private CrowdBook_Social $social;

    private CrowdBook_Frontend_Auth $frontend_auth;
    private CrowdBook_Frontend_Login $frontend_login;
    private CrowdBook_Frontend_Register $frontend_register;
    private CrowdBook_Frontend_Dashboard $frontend_dashboard;
    private CrowdBook_Frontend_Editor $frontend_editor;
    private CrowdBook_Frontend_Book_Index $frontend_book_index;
    private CrowdBook_Frontend_Books $frontend_books;

    private CrowdBook_Admin_Chapters_Page $admin_chapters;
    private CrowdBook_Admin_Users_Page $admin_users;
    private CrowdBook_Admin_Dashboard_Page $admin_dashboard;
    private CrowdBook_Admin_Books_Page $admin_books;

    public function __construct()
    {
        $this->mailer = new CrowdBook_Mailer();
        $this->books = new CrowdBook_Books();
        $this->users = new CrowdBook_Users();
        $this->auth = new CrowdBook_Auth($this->users, $this->mailer);
        $this->spam_filter = new CrowdBook_Spam_Filter();
        $this->sml_renderer = new CrowdBook_SML_Renderer();
        $this->chapters = new CrowdBook_Chapters($this->users, $this->spam_filter, $this->sml_renderer, $this->mailer);
        $this->likes = new CrowdBook_Likes($this->users, $this->chapters, $this->mailer);
        $this->social = new CrowdBook_Social($this->chapters);

        $this->frontend_auth = new CrowdBook_Frontend_Auth();
        $this->frontend_login = new CrowdBook_Frontend_Login($this->auth, $this->users);
        $this->frontend_register = new CrowdBook_Frontend_Register($this->auth, $this->users);
        $this->frontend_dashboard = new CrowdBook_Frontend_Dashboard($this->users, $this->chapters, $this->books);
        $this->frontend_editor = new CrowdBook_Frontend_Editor($this->users, $this->books, $this->chapters, $this->frontend_login);
        $this->frontend_book_index = new CrowdBook_Frontend_Book_Index($this->chapters, $this->users, $this->books, $this->sml_renderer);
        $this->frontend_books = new CrowdBook_Frontend_Books($this->books, $this->chapters, $this->sml_renderer);

        $this->admin_chapters = new CrowdBook_Admin_Chapters_Page($this->chapters, $this->books);
        $this->admin_users = new CrowdBook_Admin_Users_Page($this->users, $this->chapters);
        $this->admin_dashboard = new CrowdBook_Admin_Dashboard_Page($this->books, $this->chapters);
        $this->admin_books = new CrowdBook_Admin_Books_Page($this->books);

        add_action('init', [$this, 'bootstrap_session'], 1);
        add_action('init', [$this, 'maybe_upgrade'], 2);
        add_action('init', [$this, 'ensure_roles'], 3);
        add_action('init', [$this, 'register_rewrites']);
        add_filter('query_vars', [$this, 'register_query_vars']);
        add_action('template_redirect', [$this, 'handle_front_routes']);

        add_action('init', [$this, 'register_shortcodes']);
        add_action('wp_enqueue_scripts', [$this, 'enqueue_assets']);
        add_action('admin_menu', [$this, 'register_admin_menu']);
        add_action('wp_ajax_crowdbook_upload_image', [$this, 'ajax_upload_image']);
        add_action('wp_ajax_nopriv_crowdbook_upload_image', [$this, 'ajax_upload_image']);
        add_action('wp_ajax_crowdbook_upload_cover', [$this, 'ajax_upload_cover']);
        add_action('wp_ajax_nopriv_crowdbook_upload_cover', [$this, 'ajax_upload_cover']);

        $this->likes->register_ajax();
        $this->social->register();
    }

    public static function activate(): void
    {
        self::install_roles();
        self::create_tables();
        update_option('crowdbook_db_version', self::DB_VERSION);
        self::register_rewrite_rules();
        flush_rewrite_rules();
    }

    public static function deactivate(): void
    {
        flush_rewrite_rules();
    }

    public function bootstrap_session(): void
    {
        $this->users->ensure_session();
    }

    public function maybe_upgrade(): void
    {
        self::install_roles();

        $current = (string) get_option('crowdbook_db_version', '');
        if ($current === self::DB_VERSION) {
            return;
        }

        self::create_tables();
        self::register_rewrite_rules();
        flush_rewrite_rules();
        update_option('crowdbook_db_version', self::DB_VERSION);
    }

    public function ensure_roles(): void
    {
        self::install_roles();
    }

    public function register_rewrites(): void
    {
        self::register_rewrite_rules();
    }

    public static function register_rewrite_rules(): void
    {
        add_rewrite_rule('^login/?$', 'index.php?crowdbook_login=1', 'bottom');
        add_rewrite_rule('^register/?$', 'index.php?crowdbook_register=1', 'bottom');
        add_rewrite_rule('^dashboard/?$', 'index.php?crowdbook_dashboard=1', 'bottom');
        add_rewrite_rule('^editor/?$', 'index.php?crowdbook_editor=1', 'bottom');
        add_rewrite_rule('^books/?$', 'index.php?crowdbook_books=1', 'bottom');
        add_rewrite_rule('^book/([^/]+)/?$', 'index.php?crowdbook_book=$matches[1]', 'bottom');

        add_rewrite_rule('^crowdbook/auth/?$', 'index.php?crowdbook_auth=1', 'top');
        add_rewrite_rule('^crowdbook/login/?$', 'index.php?crowdbook_login=1', 'top');
        add_rewrite_rule('^crowdbook/register/?$', 'index.php?crowdbook_register=1', 'top');
        add_rewrite_rule('^crowdbook/dashboard/?$', 'index.php?crowdbook_dashboard=1', 'top');
        add_rewrite_rule('^crowdbook/editor/?$', 'index.php?crowdbook_editor=1', 'top');
        add_rewrite_rule('^crowdbook/books/?$', 'index.php?crowdbook_books=1', 'top');
        add_rewrite_rule('^crowdbook/book/([^/]+)/?$', 'index.php?crowdbook_book=$matches[1]', 'top');
        add_rewrite_rule('^crowdbook/chapter/([^/]+)/?$', 'index.php?crowdbook_chapter=$matches[1]', 'top');
    }

    public function register_query_vars(array $vars): array
    {
        $vars[] = 'crowdbook_auth';
        $vars[] = 'crowdbook_login';
        $vars[] = 'crowdbook_register';
        $vars[] = 'crowdbook_dashboard';
        $vars[] = 'crowdbook_editor';
        $vars[] = 'crowdbook_books';
        $vars[] = 'crowdbook_book';
        $vars[] = 'crowdbook_chapter';

        return $vars;
    }

    public function register_shortcodes(): void
    {
        add_shortcode('crowdbook_login', function (): string {
            return $this->frontend_login->render();
        });

        add_shortcode('crowdbook_register', function (): string {
            return $this->frontend_register->render();
        });

        add_shortcode('crowdbook_dashboard', function (): string {
            return $this->frontend_dashboard->render();
        });

        add_shortcode('crowdbook_editor', function (): string {
            return $this->frontend_editor->render();
        });

        add_shortcode('crowdbook_index', function (array $atts): string {
            return $this->frontend_book_index->render($atts);
        });

        add_shortcode('crowdbook_books', function (): string {
            return $this->frontend_books->render();
        });
    }

    public function enqueue_assets(): void
    {
        if (!$this->is_crowdbook_context()) {
            return;
        }

        wp_enqueue_style(
            'crowdbook-css',
            plugins_url('assets/crowdbook.css', __FILE__),
            [],
            '0.1.65'
        );

        wp_enqueue_script('sml-monaco-loader', 'https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.52.2/min/vs/loader.min.js', [], null, true);
        wp_enqueue_script(
            'crowdbook-isotope',
            'https://unpkg.com/isotope-layout@3/dist/isotope.pkgd.min.js',
            ['jquery'],
            '3.0.6',
            true
        );

        wp_enqueue_script(
            'crowdbook-js',
            plugins_url('assets/crowdbook.js', __FILE__),
            ['sml-monaco-loader', 'crowdbook-isotope'],
            '0.1.65',
            true
        );

        wp_localize_script('crowdbook-js', 'crowdbook', [
            'ajax_url' => admin_url('admin-ajax.php'),
            'nonce' => wp_create_nonce('crowdbook_like_nonce'),
            'upload_nonce' => wp_create_nonce('crowdbook_upload_image_nonce'),
            'cover_upload_nonce' => wp_create_nonce('crowdbook_upload_cover_nonce'),
            'copy_label' => __('Link kopiert', 'crowdbook'),
            'upload_ok' => __('Bild hochgeladen und eingefuegt.', 'crowdbook'),
            'upload_fail' => __('Upload fehlgeschlagen.', 'crowdbook'),
            'cover_upload_ok' => __('Cover hochgeladen.', 'crowdbook'),
            'cover_upload_fail' => __('Cover-Upload fehlgeschlagen.', 'crowdbook'),
            'cover_recommended_width' => 800,
            'cover_recommended_height' => 1200,
        ]);
    }

    public function register_admin_menu(): void
    {
        add_menu_page(
            __('CrowdBook', 'crowdbook'),
            __('CrowdBook', 'crowdbook'),
            self::CAP_MODERATE,
            'crowdbook-dashboard',
            function (): void {
                $this->admin_dashboard->render();
            },
            'dashicons-book',
            58
        );

        add_submenu_page(
            'crowdbook-dashboard',
            __('Bücher', 'crowdbook'),
            __('Bücher', 'crowdbook'),
            self::CAP_MODERATE,
            'crowdbook-books',
            function (): void {
                $this->admin_books->handle_actions();
                $this->admin_books->render();
            }
        );

        add_submenu_page(
            'crowdbook-dashboard',
            __('Kapitel Moderation', 'crowdbook'),
            __('Kapitel Moderation', 'crowdbook'),
            self::CAP_MODERATE,
            'crowdbook-chapters',
            function (): void {
                $this->admin_chapters->handle_actions();
                $this->admin_chapters->render();
            }
        );

        add_submenu_page(
            'crowdbook-dashboard',
            __('User', 'crowdbook'),
            __('User', 'crowdbook'),
            'manage_options',
            'crowdbook-users',
            function (): void {
                $this->admin_users->handle_actions();
                $this->admin_users->render();
            }
        );
    }

    public function handle_front_routes(): void
    {
        if (isset($_GET['crowdbook_logout']) && $_GET['crowdbook_logout'] === '1') {
            $this->users->clear_session_email();
            wp_safe_redirect(home_url('/'));
            exit;
        }

        if ((int) get_query_var('crowdbook_auth') === 1) {
            $token = isset($_GET['token']) ? (string) $_GET['token'] : '';
            $ok = $this->auth->validate_token($token);
            $this->frontend_auth->render_result($ok);
            exit;
        }

        if ((int) get_query_var('crowdbook_login') === 1) {
            $this->render_public_page(__('Login', 'crowdbook'), $this->frontend_login->render());
            exit;
        }

        if ((int) get_query_var('crowdbook_register') === 1) {
            $this->render_public_page(__('Registrierung', 'crowdbook'), $this->frontend_register->render());
            exit;
        }

        if ((int) get_query_var('crowdbook_dashboard') === 1) {
            $this->render_public_page(__('Dashboard', 'crowdbook'), $this->frontend_dashboard->render());
            exit;
        }

        if ((int) get_query_var('crowdbook_editor') === 1) {
            $this->render_public_page(__('Editor', 'crowdbook'), $this->frontend_editor->render());
            exit;
        }

        if ((int) get_query_var('crowdbook_books') === 1) {
            $this->render_public_page(__('Alle Bücher', 'crowdbook'), $this->frontend_books->render());
            exit;
        }

        $book = get_query_var('crowdbook_book');
        if ((!is_string($book) || $book === '') && isset($_SERVER['REQUEST_URI'])) {
            $path = parse_url((string) $_SERVER['REQUEST_URI'], PHP_URL_PATH);
            $path = is_string($path) ? trim($path, '/') : '';
            if (preg_match('#^book/([^/]+)$#', $path, $m)) {
                $book = sanitize_key((string) $m[1]);
            }
        }
        if (is_string($book) && $book !== '') {
            $book_row = $this->books->get_by_book_id($book);
            $admin_preview = isset($_GET['preview']) && $_GET['preview'] === '1' && current_user_can(self::CAP_MODERATE);
            if ($book_row && (string) $book_row->status !== 'active' && !$admin_preview) {
                global $wp_query;
                $wp_query->set_404();
                status_header(404);
                nocache_headers();
                include get_404_template();
                return;
            }
            $display_book = ($book_row && $admin_preview && $this->books->has_pending_version($book_row))
                ? $this->books->get_effective_preview($book_row)
                : $book_row;
            $page_title = $display_book ? (string) $display_book->title : __('Buch', 'crowdbook');
            $this->render_public_page($page_title, $this->frontend_book_index->render(['book' => $book]));
            exit;
        }

        $slug = get_query_var('crowdbook_chapter');
        if (is_string($slug) && $slug !== '') {
            $this->render_chapter_page($slug);
            exit;
        }
    }

    private function render_chapter_page(string $slug): void
    {
        $chapter = $this->chapters->get_by_slug($slug);
        $admin_preview = isset($_GET['preview']) && $_GET['preview'] === '1' && current_user_can(self::CAP_MODERATE);
        $is_visible = $chapter && ((string) $chapter->status === 'published' || $admin_preview);

        if (!$is_visible) {
            global $wp_query;
            $wp_query->set_404();
            status_header(404);
            nocache_headers();
            include get_404_template();
            return;
        }

        status_header(200);
        nocache_headers();

        $pending_preview = $admin_preview
            && (string) $chapter->status === 'published'
            && (string) ($chapter->pending_status ?? 'none') === 'pending'
            && trim((string) ($chapter->pending_markdown_content ?? '')) !== '';

        if ($pending_preview) {
            $preview_chapter = clone $chapter;
            $preview_chapter->title = (string) ($chapter->pending_title ?: $chapter->title);
            $preview_chapter->path_label = $chapter->pending_path_label !== null ? (string) $chapter->pending_path_label : (string) $chapter->path_label;
            $preview_chapter->markdown_content = (string) $chapter->pending_markdown_content;
            $body = $this->chapters->build_chapter_body_html($preview_chapter);
        } else {
            $static_file = $this->chapter_static_file((string) $chapter->slug);
            $use_static = (string) $chapter->status === 'published' && is_readable($static_file);
            $body = $use_static
                ? (string) file_get_contents($static_file)
                : $this->chapters->build_chapter_body_html($chapter);
        }

        $chapter_url = $this->chapters->chapter_url((string) $chapter->slug);
        $body .= '<section class="crowdbook-reader-actions">';
        $liked_user = $this->users->get_current_user();
        if ((string) $chapter->status === 'published') {
            if ($liked_user) {
                $already_liked = $this->likes->has_user_liked((int) $chapter->id, (int) $liked_user->id);
                $btn_class = $already_liked ? 'crowdbook-like-button liked' : 'crowdbook-like-button';
                $btn_disabled = $already_liked ? ' disabled="disabled" aria-disabled="true"' : '';
                $body .= '<button class="' . esc_attr($btn_class) . '" data-chapter-id="' . (int) $chapter->id . '"' . $btn_disabled . '>👍 <span class="like-count">' . (int) $chapter->like_count . '</span></button>';
                if ($already_liked) {
                    $body .= '<span class="crowdbook-like-note">' . esc_html__('Bereits geliked', 'crowdbook') . '</span>';
                }
            } else {
                $login_url = esc_url(home_url('/login'));
                $body .= '<p><a class="btn btn-primary crowdbook-nav-btn" href="' . $login_url . '">' . esc_html__('Einloggen, um zu liken', 'crowdbook') . '</a> · 👍 ' . (int) $chapter->like_count . '</p>';
            }
            $body .= $this->social->render_share_buttons((string) $chapter->title, $chapter_url);
            if ($pending_preview) {
                $body .= '<p><em>' . esc_html__('Admin-Vorschau: Du siehst die eingereichte Update-Version, noch nicht live.', 'crowdbook') . '</em></p>';
            }
        } elseif ($admin_preview) {
            $body .= '<p><em>' . esc_html__('Admin-Vorschau: Dieses Kapitel ist noch nicht veröffentlicht.', 'crowdbook') . '</em></p>';
        }
        $body .= '</section>';

        get_header();
        echo '<div id="wrapper">';
        echo '<section class="container main-content">';
        echo '<article class="white-row crowdbook-page crowdbook-reader">';
        echo '<div class="entry-content crowdbook-content">' . $body . '</div>';
        echo '</article>';
        echo '</section>';
        echo '</div>';
        get_footer();
    }

    private function chapter_static_file(string $slug): string
    {
        $uploads = wp_upload_dir();
        $basedir = is_array($uploads) && isset($uploads['basedir']) ? (string) $uploads['basedir'] : '';

        return trailingslashit($basedir) . 'crowdbook/' . sanitize_file_name($slug) . '.html';
    }

    private function is_crowdbook_context(): bool
    {
        if ((int) get_query_var('crowdbook_login') === 1 || (int) get_query_var('crowdbook_register') === 1 || (int) get_query_var('crowdbook_dashboard') === 1 || (int) get_query_var('crowdbook_editor') === 1 || (int) get_query_var('crowdbook_books') === 1) {
            return true;
        }

        $book = get_query_var('crowdbook_book');
        if (is_string($book) && $book !== '') {
            return true;
        }

        $slug = get_query_var('crowdbook_chapter');
        if (is_string($slug) && $slug !== '') {
            return true;
        }

        global $post;
        if (!($post instanceof WP_Post)) {
            return false;
        }

        $content = (string) $post->post_content;
        return str_contains($content, '[crowdbook_') || str_contains($content, '[crowdbook_register]') || str_contains($content, '[crowdbook_books]');
    }

    private function render_public_page(string $title, string $body): void
    {
        status_header(200);
        nocache_headers();

        get_header();
        echo '<div id="wrapper">';
        echo '<section class="container main-content">';
        echo '<article class="white-row crowdbook-page crowdbook-reader">';
        echo '<header class="entry-header"><h1 class="entry-title">' . esc_html($title) . '</h1></header>';
        echo '<div class="entry-content crowdbook-content">' . $body . '</div>';
        echo '</article>';
        echo '</section>';
        echo '</div>';
        get_footer();
    }

    public function ajax_upload_image(): void
    {
        check_ajax_referer('crowdbook_upload_image_nonce', 'nonce');

        $author = $this->users->get_current_user();
        if (!$author) {
            wp_send_json_error(['message' => __('Bitte zuerst per Magic Link einloggen.', 'crowdbook')], 403);
        }

        if (!isset($_FILES['image']) || !is_array($_FILES['image'])) {
            wp_send_json_error(['message' => __('Keine Datei gefunden.', 'crowdbook')], 400);
        }

        $max_size = defined('CROWDBOOK_IMAGE_MAX_BYTES')
            ? max(1024, (int) CROWDBOOK_IMAGE_MAX_BYTES)
            : 8 * 1024 * 1024;
        $file_size = isset($_FILES['image']['size']) ? (int) $_FILES['image']['size'] : 0;
        if ($file_size <= 0 || $file_size > $max_size) {
            wp_send_json_error(['message' => __('Datei ist leer oder zu gross.', 'crowdbook')], 400);
        }

        require_once ABSPATH . 'wp-admin/includes/file.php';

        $upload = wp_handle_upload($_FILES['image'], [
            'test_form' => false,
            'mimes' => [
                'jpg|jpeg|jpe' => 'image/jpeg',
                'png' => 'image/png',
                'gif' => 'image/gif',
                'webp' => 'image/webp',
                'avif' => 'image/avif',
            ],
        ]);

        if (!is_array($upload) || isset($upload['error']) || empty($upload['url'])) {
            $error = is_array($upload) && isset($upload['error']) ? (string) $upload['error'] : __('Upload fehlgeschlagen.', 'crowdbook');
            wp_send_json_error(['message' => $error], 400);
        }

        $url = esc_url_raw((string) $upload['url']);
        $alt = sanitize_file_name(pathinfo((string) ($_FILES['image']['name'] ?? 'image'), PATHINFO_FILENAME));
        if ($alt === '') {
            $alt = 'image';
        }
        $markdown = '![' . $alt . '](' . $url . ')';

        wp_send_json_success([
            'url' => $url,
            'markdown' => $markdown,
            'message' => __('Bild hochgeladen.', 'crowdbook'),
        ]);
    }

    public function ajax_upload_cover(): void
    {
        check_ajax_referer('crowdbook_upload_cover_nonce', 'nonce');

        $author = $this->users->get_current_user();
        if (!$author) {
            wp_send_json_error(['message' => __('Bitte zuerst per Magic Link einloggen.', 'crowdbook')], 403);
        }

        if (!isset($_FILES['cover']) || !is_array($_FILES['cover'])) {
            wp_send_json_error(['message' => __('Keine Datei gefunden.', 'crowdbook')], 400);
        }

        $max_size = defined('CROWDBOOK_IMAGE_MAX_BYTES')
            ? max(1024, (int) CROWDBOOK_IMAGE_MAX_BYTES)
            : 8 * 1024 * 1024;
        $file_size = isset($_FILES['cover']['size']) ? (int) $_FILES['cover']['size'] : 0;
        if ($file_size <= 0 || $file_size > $max_size) {
            wp_send_json_error(['message' => __('Datei ist leer oder zu gross.', 'crowdbook')], 400);
        }

        require_once ABSPATH . 'wp-admin/includes/file.php';
        require_once ABSPATH . 'wp-admin/includes/image.php';

        $upload = wp_handle_upload($_FILES['cover'], [
            'test_form' => false,
            'mimes' => [
                'jpg|jpeg|jpe' => 'image/jpeg',
                'png' => 'image/png',
                'webp' => 'image/webp',
                'avif' => 'image/avif',
            ],
        ]);

        if (!is_array($upload) || isset($upload['error']) || empty($upload['url']) || empty($upload['file'])) {
            $error = is_array($upload) && isset($upload['error']) ? (string) $upload['error'] : __('Upload fehlgeschlagen.', 'crowdbook');
            wp_send_json_error(['message' => $error], 400);
        }

        $target_width = defined('CROWDBOOK_COVER_WIDTH') ? max(300, (int) CROWDBOOK_COVER_WIDTH) : 800;
        $target_height = defined('CROWDBOOK_COVER_HEIGHT') ? max(450, (int) CROWDBOOK_COVER_HEIGHT) : 1200;
        $cover_url = esc_url_raw((string) $upload['url']);
        $cover_path = (string) $upload['file'];
        $resize_note = '';

        $editor = wp_get_image_editor($cover_path);
        if (!is_wp_error($editor)) {
            $source_size = $editor->get_size();
            $resized = $editor->resize($target_width, $target_height, true);

            if (!is_wp_error($resized)) {
                $saved = $editor->save($cover_path);
                if (!is_wp_error($saved) && is_array($saved) && !empty($saved['path'])) {
                    $cover_path = (string) $saved['path'];
                    $uploads = wp_upload_dir();
                    if (
                        is_array($uploads)
                        && !empty($uploads['basedir'])
                        && !empty($uploads['baseurl'])
                        && str_starts_with($cover_path, (string) $uploads['basedir'])
                    ) {
                        $relative_path = ltrim(substr($cover_path, strlen((string) $uploads['basedir'])), '/');
                        $cover_url = trailingslashit((string) $uploads['baseurl']) . $relative_path;
                    }
                }
            } elseif (is_array($source_size) && isset($source_size['width'], $source_size['height'])) {
                if ((int) $source_size['width'] < $target_width || (int) $source_size['height'] < $target_height) {
                    $resize_note = __('Hinweis: Bild ist kleiner als die empfohlene Cover-Aufloesung.', 'crowdbook');
                }
            }
        }

        $size = @getimagesize($cover_path);
        $width = is_array($size) && isset($size[0]) ? (int) $size[0] : 0;
        $height = is_array($size) && isset($size[1]) ? (int) $size[1] : 0;

        wp_send_json_success([
            'url' => $cover_url,
            'width' => $width,
            'height' => $height,
            'message' => $resize_note !== '' ? $resize_note : __('Cover hochgeladen und verarbeitet.', 'crowdbook'),
            'recommended_width' => $target_width,
            'recommended_height' => $target_height,
        ]);
    }

    public static function create_tables(): void
    {
        global $wpdb;

        require_once ABSPATH . 'wp-admin/includes/upgrade.php';

        $charset_collate = $wpdb->get_charset_collate();
        $users = $wpdb->prefix . 'crowdbook_users';
        $tokens = $wpdb->prefix . 'crowdbook_magic_tokens';
        $books = $wpdb->prefix . 'crowdbook_books';
        $chapters = $wpdb->prefix . 'crowdbook_chapters';
        $likes = $wpdb->prefix . 'crowdbook_likes';

        $sql_users = "CREATE TABLE {$users} (
            id INT AUTO_INCREMENT PRIMARY KEY,
            email VARCHAR(200) NOT NULL UNIQUE,
            display_name VARCHAR(200) NOT NULL,
            bio TEXT,
            status ENUM('active', 'banned') DEFAULT 'active',
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        ) {$charset_collate};";

        $sql_tokens = "CREATE TABLE {$tokens} (
            id INT AUTO_INCREMENT PRIMARY KEY,
            email VARCHAR(200) NOT NULL,
            token VARCHAR(64) NOT NULL UNIQUE,
            expires_at DATETIME NOT NULL,
            used TINYINT DEFAULT 0,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        ) {$charset_collate};";

        $sql_books = "CREATE TABLE {$books} (
            id INT AUTO_INCREMENT PRIMARY KEY,
            book_id VARCHAR(100) NOT NULL UNIQUE,
            title VARCHAR(300) NOT NULL,
            description TEXT,
            prologue_markdown LONGTEXT,
            cover_image_url VARCHAR(600),
            status ENUM('pending', 'active', 'archived') DEFAULT 'pending',
            pending_title VARCHAR(300) NULL,
            pending_description TEXT NULL,
            pending_prologue_markdown LONGTEXT NULL,
            pending_cover_image_url VARCHAR(600) NULL,
            pending_status ENUM('none', 'draft', 'pending', 'rejected') DEFAULT 'none',
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
        ) {$charset_collate};";

        $sql_chapters = "CREATE TABLE {$chapters} (
            id INT AUTO_INCREMENT PRIMARY KEY,
            book_id VARCHAR(100) NOT NULL,
            author_id INT NOT NULL,
            title VARCHAR(300) NOT NULL,
            slug VARCHAR(300) NOT NULL UNIQUE,
            path_label VARCHAR(100),
            markdown_content LONGTEXT,
            status ENUM('draft', 'pending', 'published', 'rejected') DEFAULT 'draft',
            spam_score FLOAT DEFAULT 0.0,
            pending_title VARCHAR(300) NULL,
            pending_path_label VARCHAR(100) NULL,
            pending_markdown_content LONGTEXT NULL,
            pending_status ENUM('none', 'draft', 'pending', 'rejected') DEFAULT 'none',
            pending_spam_score FLOAT DEFAULT 0.0,
            like_count INT DEFAULT 0,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            published_at DATETIME NULL,
            KEY author_id (author_id)
        ) {$charset_collate};";

        $sql_likes = "CREATE TABLE {$likes} (
            id INT AUTO_INCREMENT PRIMARY KEY,
            chapter_id INT NOT NULL,
            user_id INT NULL,
            fingerprint VARCHAR(64) NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            UNIQUE KEY unique_like_user (chapter_id, user_id),
            KEY chapter_id (chapter_id),
            KEY user_id (user_id)
        ) {$charset_collate};";

        dbDelta($sql_users);
        dbDelta($sql_tokens);
        dbDelta($sql_books);
        dbDelta($sql_chapters);
        dbDelta($sql_likes);

        $books_repo = new CrowdBook_Books();
        $books_repo->purge_legacy_seed_book();
    }

    private static function install_roles(): void
    {
        $moderator = get_role('crowdbook_moderator');
        if (!$moderator) {
            add_role('crowdbook_moderator', 'CrowdBook Moderator', [
                'read' => true,
                self::CAP_MODERATE => true,
            ]);
        } else {
            $moderator->add_cap(self::CAP_MODERATE);
        }

        $admin = get_role('administrator');
        if ($admin) {
            $admin->add_cap(self::CAP_MODERATE);
        }
    }
}

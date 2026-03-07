<?php
if (!defined('ABSPATH')) {
    exit;
}

while (have_posts()) :
    the_post();
    $html = SML_Pages_Plugin::get_rendered_for_post(get_the_ID());
    ?>
<!doctype html>
<html <?php language_attributes(); ?>>
<head>
    <meta charset="<?php bloginfo('charset'); ?>" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <?php wp_head(); ?>
</head>
<body <?php body_class('sml-canvas-mode'); ?>>
<?php wp_body_open(); ?>

<main class="sml-canvas-root" aria-label="SML Canvas">
    <?php echo wp_kses_post($html); ?>
</main>

<?php wp_footer(); ?>
</body>
</html>
    <?php
endwhile;

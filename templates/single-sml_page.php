<?php
if (!defined('ABSPATH')) {
    exit;
}

get_header();

while (have_posts()) :
    the_post();
    $html = SML_Pages_Plugin::get_rendered_for_post(get_the_ID());
    ?>
    <main class="container sml-page-shell">
        <article>
            <header>
                <h1><?php the_title(); ?></h1>
            </header>
            <section class="sml-render-root">
                <?php echo wp_kses_post($html); ?>
            </section>
        </article>
    </main>
    <?php
endwhile;

get_footer();

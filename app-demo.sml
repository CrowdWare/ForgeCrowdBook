App {
    name: "ForgeCrowdBook"
    base_url: "http://localhost:8090"
    db: "./data/crowdbook.db"
    port: "8090"

    # Secrets: leave blank here and set via environment variables instead.
    # export FCB_SESSION_SECRET="<random-string-min-32-chars>"
    # export FCB_ADMIN_EMAIL="you@example.com"
    # export FCB_SMTP_USER="your-smtp-user"
    # export FCB_SMTP_PASS="your-smtp-password"
    session_secret: ""
    admin_email: ""

    SMTP {
        host: "smtp.example.com"
        port: "587"
        user: ""
        pass: ""
        from: "noreply@example.com"
    }
}

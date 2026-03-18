App {
    name: "ForgeCrowdBook (Demo)"
    base_url: "http://localhost:8090"
    db: "./data/crowdbook-demo.db"
    port: "8090"
    session_secret: "demo-secret-change-me"
    admin_email: "admin@example.com"

    SMTP {
        host: "smtp.example.com"
        port: "587"
        user: "smtp-user"
        pass: "smtp-password"
        from: "noreply@example.com"
    }
}

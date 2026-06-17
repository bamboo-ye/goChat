package main

// This package is intentionally inert. Database migrations run during
// application startup and automated tests must not mutate shared databases.
func main() {}

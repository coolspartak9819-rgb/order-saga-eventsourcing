# Security Notes

- SSH access should use keys, disable password authentication and be restricted
  by firewall rules or VPN.
- Production secrets must come from GitLab CI protected variables or a secret
  manager, not from the repository.
- TLS certificates should be issued and renewed by Certbot or the selected
  cloud certificate service.
- The cache purge endpoint must not be publicly exposed without authentication.
- Container images should run as non-root where the base image and process
  model allow it, and be scanned before release.

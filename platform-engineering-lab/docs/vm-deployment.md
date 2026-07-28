# Debian VM Deployment

## Prepare access

Create a Debian 12 VM, add an SSH key and put its address in
`infra/ansible/inventory.example`. Do not expose MySQL, Redis, Prometheus or
Grafana directly to the public internet.

## Install and deploy

```bash
ansible-galaxy collection install community.general
ansible-playbook -i infra/ansible/inventory.example infra/ansible/site.yml
ansible-playbook -i infra/ansible/inventory.example infra/ansible/deploy.yml
```

The playbook installs Docker, enables the service, configures the firewall and
starts the production Compose overlay. The systemd unit can be copied to
`/etc/systemd/system/platform-engineering-lab.service` and enabled with:

```bash
systemctl daemon-reload
systemctl enable --now platform-engineering-lab
```

## Release and rollback

Deployments should be made from a Git tag or commit SHA. Set `APP_VERSION` in
the environment and run the smoke check after the deployment. Roll back by
starting the previous image tag and recording the incident in the runbook.
